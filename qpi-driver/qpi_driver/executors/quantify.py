import copy
import importlib
from contextlib import suppress
from pathlib import Path
from typing import Any

import numpy as np
import qiskit.circuit
import xarray as xr
import yaml
from qiskit import QuantumCircuit
from qiskit.circuit import library as qiskit_library

from qpi_driver.compat.quantify import (
    CNOT,
    CZ,
    IS_QUANTIFY_INSTALLED,
    Cluster,
    ClusterComponent,
    ClusterDescription,
    ClusterType,
    H,
    Instrument,
    InstrumentCoordinator,
    InstrumentModule,
    InstrumentType,
    Measure,
    Operation,
    ParameterBase,
    QbloxHardwareCompilationConfig,
    QuantumDevice,
    Reset,
    Rxy,
    Rz,
    S,
    Schedule,
    SDagger,
    SerialCompiler,
    T,
    TDagger,
    X,
    Y,
    Z,
    set_datadir,
)
from qpi_driver.executors.base import Executor, JobPayload

from .utils.qiskit import load_qasm

_DEVICE_ELEMENT_TYPE_PROP = "element_type"


class QuantifyExecutor(Executor):
    """Executor subclass for interacting with Quantify-scheduler acquisition backends."""

    def __new__(cls, *args, **kwargs):
        if not IS_QUANTIFY_INSTALLED:
            raise ImportError(
                "quantify-scheduler is not installed. Install the [quantify] extra to use QuantifyExecutor."
            )
        return super().__new__(cls)

    def __init__(
        self,
        name: str = "quantify",
        quantify_hardware_config: QbloxHardwareCompilationConfig | Path | dict = Path(
            "quantify.hardware.json"
        ),
        quantify_device_config: Path | dict = Path("quantify.device.json"),
        is_dummy: bool = False,
        data_dir: Path = Path("data"),
        acquisition_timeout: int = 10,
        **kwargs: Any,
    ) -> None:
        """Initialize the QuantifyExecutor.

        Args:
            name: the name of the executor
            quantify_hardware_config: Hardware-layer configuration dictionary, file path, or config as dict.
            quantify_device_config: Device-layer configuration dictionary, file path or config as dict
            is_dummy: If True, uses a dummy Cluster instrument.
            data_dir: Directory to where data is temporarily stored.
            acquisition_timeout: Timeout in seconds to wait for acquisition.
            **kwargs: Arbitrary keyword arguments passed to the base class.
        """
        super().__init__(name, **kwargs)
        set_datadir(data_dir)
        # Clean up any previously registered instruments to avoid name collision errors in QCoDeS
        with suppress(Exception):
            Instrument.close_all()

        self._is_dummy = is_dummy
        self._acquisition_timeout = acquisition_timeout
        hardware_config = _load_quantify_hardware_config(quantify_hardware_config)
        self._hardware_config = hardware_config
        self._device = _load_quantum_device(name=name, config=quantify_device_config)
        self._instrument_coordinator = _load_instrument_coordinator(
            f"{name}_ic", hardware_config=hardware_config, is_dummy=is_dummy
        )
        self._device.hardware_config(hardware_config)
        self._compiler = SerialCompiler(
            name=f"{name}_compiler", quantum_device=self._device
        )

    @property
    def hardware_config(self) -> QbloxHardwareCompilationConfig:
        return self._hardware_config

    def execute(self, payload: JobPayload) -> xr.Dataset:
        """Execute quantum instructions using the Quantify scheduler.

        The acquisition protocol is selected based on ``payload.meas_level``:

        * ``meas_level=0`` → ``Trace`` (raw waveform)
        * ``meas_level=1`` → ``SSBIntegrationComplex`` (kerneled IQ)
        * ``meas_level=2`` → ``ThresholdedAcquisition`` if threshold params
          are configured on the device elements, else ``SSBIntegrationComplex``
          (software discrimination deferred to ``process_result()``).

        Args:
            payload: JobPayload specifying shots, circuits, meas_level, etc.

        Returns:
            xr.Dataset: Raw acquisition dataset.
        """
        # Parse QASM circuit
        circuit = load_qasm(payload.qasm)
        shots = payload.shots
        n_qubits = circuit.num_qubits

        # Determine acquisition protocol and parameters from meas_level and payload overrides
        acq_protocol, acq_kwargs = self._resolve_acq_protocol(payload)

        # Translate Qiskit QuantumCircuit to Quantify Schedule
        schedule = Schedule(name=payload.id, repetitions=shots)
        acq_indices = {}

        for instruction in circuit.data:
            parsed_ops = _to_quantify_gates(
                circuit=circuit,
                instruction=instruction,
                acq_indices=acq_indices,
                acq_protocol=acq_protocol,
                acq_kwargs=acq_kwargs,
            )
            for op in parsed_ops:
                schedule.add(op)

        compiled_sched = self._compiler.compile(schedule=schedule)

        self._instrument_coordinator.prepare(compiled_sched)
        self._instrument_coordinator.start()
        self._instrument_coordinator.wait_done(timeout_sec=self._acquisition_timeout)
        dataset = self._instrument_coordinator.retrieve_acquisition()
        dataset.attrs.update(
            {
                "shots": shots,
                "n_qubits": n_qubits,
                "backend": self.name,
                "meas_level": payload.meas_level,
                "meas_return": payload.meas_return,
                "acq_protocol": acq_protocol,
            }
        )
        if "acq_rotation" in acq_kwargs:
            dataset.attrs["acq_rotation"] = acq_kwargs["acq_rotation"]
        if "acq_threshold" in acq_kwargs:
            dataset.attrs["acq_threshold"] = acq_kwargs["acq_threshold"]
        return dataset

    def _resolve_acq_protocol(self, payload: JobPayload) -> tuple[str, dict]:
        """Determine the quantify-scheduler acquisition protocol for the given meas_level.

        Args:
            payload: JobPayload specifying meas_level, acq_threshold, acq_rotation, etc.

        Returns:
            Tuple of (protocol_name, extra_kwargs_for_Measure).
        """
        if payload.meas_level == 0:
            return "Trace", {}

        if payload.meas_level == 1:
            return "SSBIntegrationComplex", {}

        # meas_level == 2: try ThresholdedAcquisition if device or payload has threshold params
        dev_params = self._get_threshold_params()
        if payload.acq_rotation is not None:
            dev_params["acq_rotation"] = payload.acq_rotation
        if payload.acq_threshold is not None:
            dev_params["acq_threshold"] = payload.acq_threshold

        if "acq_rotation" in dev_params and "acq_threshold" in dev_params:
            return "ThresholdedAcquisition", dev_params

        # Fallback to SSBIntegrationComplex + software discrimination in process_result()
        return "SSBIntegrationComplex", dev_params

    def _get_threshold_params(self) -> dict:
        """Extract acq_threshold and acq_rotation from the first device element that has them.

        Returns:
            Dict with 'acq_rotation' and 'acq_threshold' if found, else empty dict.
        """
        for element_name in self._device.elements():
            element = self._device.get_element(element_name)
            try:
                threshold = element.measure.acq_threshold()
                rotation = element.measure.acq_rotation()
                if threshold is not None and rotation is not None:
                    return {
                        "acq_threshold": threshold,
                        "acq_rotation": rotation,
                    }
            except (AttributeError, KeyError):
                continue
        return {}

    def process_result(self, dataset: xr.Dataset, job_id: str) -> dict:
        """Convert a quantify-scheduler acquisition dataset into a Qiskit-compatible result dict.

        Handles all meas_levels:
        - meas_level=0 (Trace): Returns raw complex waveform data as [[real, imag], ...] per time sample.
        - meas_level=1 (SSBIntegrationComplex): Returns IQ values as [[real, imag]] per shot per qubit.
        - meas_level=2 with ThresholdedAcquisition: Aggregates 0/1 values into counts dict.
        - meas_level=2 with SSBIntegrationComplex: Performs software discrimination using
          acq_threshold and acq_rotation from the device config.

        Args:
            dataset: xr.Dataset from execute().
            job_id: Unique job ID.

        Returns:
            dict: Qiskit-compatible result dict.
        """
        from qpi_driver.executors.utils.result import build_qiskit_result

        meas_level = int(dataset.attrs.get("meas_level", 2))
        meas_return = str(dataset.attrs.get("meas_return", "single"))
        acq_protocol = str(dataset.attrs.get("acq_protocol", "SSBIntegrationComplex"))
        backend = dataset.attrs.get("backend", self.name)

        # Handle multi-circuit datasets
        if "circuit_index" in dataset.dims:
            circuit_results = []
            for ci in range(dataset.sizes["circuit_index"]):
                sub_ds = dataset.isel(circuit_index=ci)
                sub_ds.attrs.update(dataset.attrs)
                circuit_results.append(
                    self._single_dataset_to_result(
                        sub_ds, meas_level, meas_return, acq_protocol
                    )
                )
            return build_qiskit_result(circuit_results, job_id, backend)

        single = self._single_dataset_to_result(
            dataset, meas_level, meas_return, acq_protocol
        )
        return build_qiskit_result([single], job_id, backend)

    def _single_dataset_to_result(
        self, dataset: xr.Dataset, meas_level: int, meas_return: str, acq_protocol: str
    ) -> dict:
        """Extract result data from a single-circuit quantify dataset."""
        qubit_vars, q0_key, shots = self._extract_qubit_vars(dataset)
        if not qubit_vars:
            return {"raw": str(dataset), "shots": 0}

        if meas_level == 0:
            return self._process_meas_level_0(dataset, qubit_vars, shots)
        if meas_level == 1:
            return self._process_meas_level_1(dataset, qubit_vars, shots, meas_return)
        return self._process_meas_level_2(dataset, qubit_vars, shots, acq_protocol)

    def _extract_qubit_vars(self, dataset: xr.Dataset) -> tuple[list[int], str, int]:
        """Identify qubit variables (integer-named data vars) and shots."""
        qubit_vars = []
        for var in dataset.data_vars:
            try:
                qubit_vars.append(int(var))
            except ValueError:
                pass
        if not qubit_vars:
            return [], "", 0
        qubit_vars.sort()
        q0_key = qubit_vars[0] if qubit_vars[0] in dataset else str(qubit_vars[0])
        shots = int(dataset.attrs.get("shots", len(dataset[q0_key])))
        return qubit_vars, q0_key, shots

    def _process_meas_level_0(
        self, dataset: xr.Dataset, qubit_vars: list[int], shots: int
    ) -> dict:
        """Extract raw complex trace data (meas_level=0)."""
        memory: list[list[list[float]]] = []
        for q_idx in qubit_vars:
            var_key = q_idx if q_idx in dataset else str(q_idx)
            trace = dataset[var_key].values.flatten()
            qubit_trace = [[float(v.real), float(v.imag)] for v in trace]
            memory.append(qubit_trace)
        return {"memory": memory, "shots": shots}

    def _process_meas_level_1(
        self, dataset: xr.Dataset, qubit_vars: list[int], shots: int, meas_return: str
    ) -> dict:
        """Extract integrated IQ memory (meas_level=1)."""
        from qpi_driver.executors.utils.result import iq_memory_avg

        q0_key = qubit_vars[0] if qubit_vars[0] in dataset else str(qubit_vars[0])
        num_samples = len(dataset[q0_key])
        memory = []
        for s in range(num_samples):
            shot_iq = []
            for q_idx in qubit_vars:
                var_key = q_idx if q_idx in dataset else str(q_idx)
                val = dataset[var_key].values[s]
                shot_iq.append([float(val.real), float(val.imag)])
            memory.append(shot_iq)

        if meas_return == "avg" and memory:
            memory = iq_memory_avg(memory, len(qubit_vars))
        return {"memory": memory, "shots": shots}

    def _process_meas_level_2(
        self, dataset: xr.Dataset, qubit_vars: list[int], shots: int, acq_protocol: str
    ) -> dict:
        """Extract classified counts (meas_level=2) performing software discrimination if needed."""
        q0_key = qubit_vars[0] if qubit_vars[0] in dataset else str(qubit_vars[0])
        num_samples = len(dataset[q0_key])
        n_qubits = len(qubit_vars)
        counts_dict: dict[str, int] = {}

        if acq_protocol == "ThresholdedAcquisition":
            for s in range(num_samples):
                state_chars = []
                for q_idx in reversed(qubit_vars):
                    var_key = q_idx if q_idx in dataset else str(q_idx)
                    val = dataset[var_key].values[s]
                    state_chars.append(str(int(val.real)))
                state_str = "".join(state_chars)
                counts_dict[state_str] = counts_dict.get(state_str, 0) + 1
        else:
            # SSBIntegrationComplex software discrimination.
            # Check dataset.attrs first, fallback to device config, default to 0.0.
            acq_rotation = dataset.attrs.get("acq_rotation")
            acq_threshold = dataset.attrs.get("acq_threshold")
            if acq_rotation is None or acq_threshold is None:
                dev_params = self._get_threshold_params()
                acq_rotation = (
                    dev_params.get("acq_rotation", 0.0)
                    if acq_rotation is None
                    else acq_rotation
                )
                acq_threshold = (
                    dev_params.get("acq_threshold", 0.0)
                    if acq_threshold is None
                    else acq_threshold
                )

            rot_rad = np.radians(acq_rotation)
            for s in range(num_samples):
                state_chars = []
                for q_idx in reversed(qubit_vars):
                    var_key = q_idx if q_idx in dataset else str(q_idx)
                    val = dataset[var_key].values[s]
                    rotated = val * np.exp(1j * rot_rad)
                    state_chars.append("1" if rotated.real > acq_threshold else "0")
                state_str = "".join(state_chars)
                counts_dict[state_str] = counts_dict.get(state_str, 0) + 1

        if num_samples == 1 and shots > 1:
            state_str = list(counts_dict.keys())[0]
            counts_dict = {state_str: shots}

        # Pad with 0 counts for all 2^N states
        states_list = [format(i, f"0{n_qubits}b") for i in range(2**n_qubits)]
        for st in states_list:
            if st not in counts_dict:
                counts_dict[st] = 0

        return {"counts": counts_dict, "shots": shots}

    def close(self) -> None:
        """Release resources."""
        for component in self._instrument_coordinator.components:
            self._instrument_coordinator.remove_component(component.name)
            component.close()

        with suppress(Exception):
            self._instrument_coordinator.close()


def _to_quantify_gates(
    circuit: QuantumCircuit,
    instruction: qiskit.circuit.CircuitInstruction,
    acq_indices: dict[int, int],
    acq_protocol: str = "SSBIntegrationComplex",
    acq_kwargs: dict | None = None,
) -> list[Operation]:
    """Converts a qiskit Instruction to Quantify gate operations.

    Args:
        circuit: The circuit in which the instruction is.
        instruction: The instruction to convert.
        acq_indices: A mapping of qubit and current acquisitions/measurements done on said qubit in circuit..

    Returns:
        list of quantify Operations

    Raises:
        ValueError: if the gate is not supported by QuantifyExecutor
    """
    gate = instruction.operation
    name = instruction.name
    params = instruction.params

    qubit_indices = [circuit.find_bit(q).index for q in instruction.qubits]
    qubits = [f"q{idx}" for idx in qubit_indices]

    if isinstance(gate, qiskit_library.Reset):
        return [Reset(q) for q in qubits]

    if isinstance(gate, qiskit_library.XGate):
        return [X(q) for q in qubits]

    if isinstance(gate, qiskit_library.YGate):
        return [Y(q) for q in qubits]

    if isinstance(gate, qiskit_library.ZGate):
        return [Z(q) for q in qubits]

    if isinstance(gate, qiskit_library.HGate):
        return [H(q) for q in qubits]

    if isinstance(gate, qiskit_library.SGate):
        return [S(q) for q in qubits]

    if isinstance(gate, qiskit_library.SdgGate):
        return [SDagger(q) for q in qubits]

    if isinstance(gate, qiskit_library.TGate):
        return [T(q) for q in qubits]

    if isinstance(gate, qiskit_library.TdgGate):
        return [TDagger(q) for q in qubits]

    if isinstance(gate, qiskit_library.SXGate):
        return [Rxy(theta=90.0, phi=0.0, qubit=q) for q in qubits]

    if isinstance(gate, qiskit_library.SXdgGate):
        return [Rxy(theta=-90.0, phi=0.0, qubit=q) for q in qubits]

    if isinstance(gate, qiskit_library.CXGate):
        return [CNOT(qC=c, qT=t) for c, t in zip(qubits[0::2], qubits[1::2])]

    if isinstance(gate, qiskit_library.CZGate):
        return [CZ(qC=c, qT=t) for c, t in zip(qubits[0::2], qubits[1::2])]

    if isinstance(gate, qiskit_library.SwapGate):
        # Expands each control/target pair into 3 CNOT operations sequentially
        return [
            gate
            for c, t in zip(qubits[0::2], qubits[1::2])
            for gate in [CNOT(qC=c, qT=t), CNOT(qC=t, qT=c), CNOT(qC=c, qT=t)]
        ]

    if isinstance(gate, qiskit_library.RXGate):
        theta_deg = float(np.degrees(params[0]))
        return [Rxy(theta_deg, 0, q) for q in qubits]

    if isinstance(gate, qiskit_library.RYGate):
        theta_deg = float(np.degrees(params[0]))
        return [Rxy(theta_deg, 90, q) for q in qubits]

    if isinstance(gate, (qiskit_library.RZGate, qiskit_library.PhaseGate)):
        theta_deg = float(np.degrees(params[0]))
        return [Rz(theta_deg, q) for q in qubits]

    elif isinstance(gate, qiskit_library.UGate):
        theta_deg = float(np.degrees(params[0]))
        phi_deg = float(np.degrees(params[1]))
        lam_deg = float(np.degrees(params[2]))

        # Flattening a 1-to-3 sequence per qubit using a nested list comprehension
        return [
            gate
            for q in qubits
            for gate in [
                Rz(theta=lam_deg, qubit=q),
                Rxy(theta=theta_deg, phi=90.0, qubit=q),
                Rz(theta=phi_deg, qubit=q),
            ]
        ]

    if isinstance(gate, qiskit_library.Measure):
        result = []
        extra = acq_kwargs or {}
        for idx in qubit_indices:
            acq_idx = acq_indices.get(idx, 0)
            # Use unique acq_channel per qubit to avoid overlaps
            result.append(
                Measure(
                    f"q{idx}",
                    acq_channel=idx,
                    acq_index=acq_idx,
                    acq_protocol=acq_protocol,
                    **extra,
                )
            )
            # update the measurement count for that qubit to allow for multiple measurements on a qubit
            acq_indices[idx] = acq_idx + 1
        return result

    if isinstance(gate, qiskit_library.Barrier):
        return []

    raise ValueError(f"Gate '{name}' is not supported by QuantifyExecutor")


def _load_quantify_hardware_config(
    data: QbloxHardwareCompilationConfig | Path | dict,
) -> QbloxHardwareCompilationConfig:
    """Load quantify hardware-layer config from the given data

    Args:
        data: the data in form of ``QbloxHardwareCompilationConfig`` or the path to the config file or the dict
            version from which ``QbloxHardwareCompilationConfig`` can be constructed.

    Returns:
        the parsed QbloxHardwareCompilationConfig

    Raises:
        ValidationError: if data or the file at 'data' is invalid QbloxHardwareCompilationConfig
    """
    if isinstance(data, Path):
        with open(data, "r") as file:
            data: dict = yaml.safe_load(file)

    return QbloxHardwareCompilationConfig.model_validate(data)


def _load_quantum_device(name: str, config: Path | dict) -> QuantumDevice:
    """Load quantify device-layer config from the given data and returns a QuantumDevice

    Args:
        name: the name of the device
        config: the data in form of the path to the config file or the dict
            from which ``QuantumDevice`` can be constructed.

    Returns:
        the parsed QuantumDevice

    Raises:
        TypeError: if a device element's attribute is not callable
            yet it is configured like it is a parameter
        AttributeError: if a device element does not have the attribute
            yet it is being configured as if it has
    """
    if isinstance(config, Path):
        with open(config, "r") as file:
            config: dict = yaml.safe_load(file)
    else:
        config = copy.deepcopy(config)

    quantum_device = QuantumDevice(name=name)

    for element_name, element_data in config.items():  # type: str, dict
        element_type = element_data.pop(_DEVICE_ELEMENT_TYPE_PROP, None)
        if not element_type:
            raise ValueError(
                f"Element '{element_name}' is missing a '{_DEVICE_ELEMENT_TYPE_PROP}' specification."
            )

        try:
            module_path, element_cls_name = element_type.rsplit(".", 1)
        except ValueError:
            raise ValueError(
                f"Element type '{element_type}' is not a valid full import path for element class."
            )

        module = importlib.import_module(module_path)
        element_class = getattr(module, element_cls_name)

        element_instance = element_class(element_name)
        _apply_parameters(element_instance, element_data)

        quantum_device.add_element(element_instance)

    return quantum_device


def _to_num(value: Any) -> Any:
    """Convert a string value to float or int if it represents a number."""
    if isinstance(value, str):
        try:
            val = float(value)
            return int(val) if val.is_integer() else val
        except ValueError:
            pass
    return value


def _apply_parameters(obj: InstrumentModule | ParameterBase, data: dict):
    """Helper to recursively map dictionaries onto QCoDeS submodules/parameters

    Args:
        obj: the instrument module or parameter
        data: the data to apply to this module or parameter

    Raises:
        TypeError: if obj is not callable yet data is not a dict
        AttributeError: if obj does not have the attribute set on it in the data
    """
    if not isinstance(data, dict):
        data = _to_num(data)
        try:
            obj(data)
        except TypeError as exp:
            raise TypeError(
                f"{obj} is not a Parameter yet value {data} passed is not a dict"
            ) from exp

    else:
        for key, value in data.items():
            try:
                attribute = getattr(obj, key)
            except AttributeError as exp:
                raise AttributeError(f"{obj} has no attribute '{key}'") from exp

            _apply_parameters(attribute, value)


def _load_instrument_coordinator(
    name: str, hardware_config: QbloxHardwareCompilationConfig, is_dummy: bool = False
) -> InstrumentCoordinator:
    """Loads the instrument coordinator from the given hardware configuration

    Args:
        name: the name of the instrument coordinator
        hardware_config: the QbloxHardwareCompilationConfig of the setup.
        is_dummy: whether this setup is dummy or not.

    Returns:
        the parsed InstrumentCoordinator
    """
    coordinator = InstrumentCoordinator(name=name)
    hardware_description = hardware_config.hardware_description

    for instrument_name, cfg in hardware_description.items():
        if isinstance(cfg, ClusterDescription):
            cluster_ip = cfg.ip
            dummy_cfg = None
            if is_dummy:
                dummy_cfg = {
                    k: _to_cluster_type(v.instrument_type)
                    for k, v in cfg.modules.items()
                }

            cluster = Cluster(
                name=instrument_name, identifier=cluster_ip, dummy_cfg=dummy_cfg
            )
            cluster_component = ClusterComponent(cluster)
            coordinator.add_component(cluster_component)

    return coordinator


def _to_cluster_type(value: InstrumentType) -> ClusterType:
    """Converts the given InstrumentType to ClusterType
    Args:
        value: the InstrumentType to convert
    Returns:
        the corresponding ClusterType
    """
    return getattr(ClusterType, f"CLUSTER_{value}")
