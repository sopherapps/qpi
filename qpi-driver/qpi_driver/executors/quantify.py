import importlib
from contextlib import suppress
from pathlib import Path
from typing import Any

import xarray as xr
import yaml

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
    ParameterBase,
    QbloxHardwareCompilationConfig,
    QuantumDevice,
    Reset,
    Rxy,
    Rz,
    Schedule,
    SerialCompiler,
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
        name: str,
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
        self._device = _load_quantum_device(name=name, config=quantify_device_config)
        self._instrument_coordinator = _load_instrument_coordinator(
            f"{name}_ic", hardware_config=hardware_config, is_dummy=is_dummy
        )
        self._device.hardware_config(hardware_config)
        self._compiler = SerialCompiler(
            name=f"{name}_compiler", quantum_device=self._device
        )

    def execute(self, payload: JobPayload) -> xr.Dataset:
        """Execute quantum instructions using the Quantify scheduler.

        Args:
            payload: JobPayload specifying n_qubits, shots, and qasm.

        Returns:
            xr.Dataset: Standardised counts/frequencies dataset.
        """
        # Parse QASM circuit
        circuit = load_qasm(payload.qasm)

        n_qubits = circuit.num_qubits
        shots = payload.shots

        # Translate Qiskit QuantumCircuit to Quantify Schedule
        schedule = Schedule(name=payload.id, repetitions=shots)
        acq_indices = {}

        for instruction in circuit.data:
            gate = instruction.operation
            qubits = instruction.qubits
            qubit_indices = [circuit.find_bit(q).index for q in qubits]

            name = gate.name.lower()
            if name == "reset":
                for idx in qubit_indices:
                    schedule.add(Reset(f"q{idx}"))
            elif name == "x":
                for idx in qubit_indices:
                    schedule.add(X(f"q{idx}"))
            elif name == "y":
                for idx in qubit_indices:
                    schedule.add(Y(f"q{idx}"))
            elif name == "z":
                for idx in qubit_indices:
                    schedule.add(Z(f"q{idx}"))
            elif name == "h":
                for idx in qubit_indices:
                    schedule.add(H(f"q{idx}"))
            elif name in ("cx", "cnot"):
                ctrl = f"q{qubit_indices[0]}"
                tgt = f"q{qubit_indices[1]}"
                schedule.add(CNOT(ctrl, tgt))
            elif name == "cz":
                ctrl = f"q{qubit_indices[0]}"
                tgt = f"q{qubit_indices[1]}"
                schedule.add(CZ(ctrl, tgt))
            elif name == "rx":
                import numpy as np

                theta_deg = float(np.degrees(gate.params[0]))
                for idx in qubit_indices:
                    schedule.add(Rxy(theta_deg, 0, f"q{idx}"))
            elif name == "ry":
                import numpy as np

                theta_deg = float(np.degrees(gate.params[0]))
                for idx in qubit_indices:
                    schedule.add(Rxy(theta_deg, 90, f"q{idx}"))
            elif name == "rz":
                import numpy as np

                theta_deg = float(np.degrees(gate.params[0]))
                for idx in qubit_indices:
                    schedule.add(Rz(theta_deg, f"q{idx}"))
            elif name == "measure":
                for idx in qubit_indices:
                    acq_idx = acq_indices.get(idx, 0)
                    # Use unique acq_channel per qubit to avoid overlaps
                    schedule.add(Measure(f"q{idx}", acq_channel=idx, acq_index=acq_idx))
                    acq_indices[idx] = acq_idx + 1
            elif name == "barrier":
                pass
            else:
                raise ValueError(f"Gate '{name}' is not supported by QuantifyExecutor")

        compiled_sched = self._compiler.compile(schedule=schedule)

        self._instrument_coordinator.prepare(compiled_sched)
        self._instrument_coordinator.start()
        self._instrument_coordinator.wait_done(timeout_sec=self._acquisition_timeout)
        return self._instrument_coordinator.retrieve_acquisition()

    def close(self) -> None:
        """Release resources."""
        for component in self._instrument_coordinator.components:
            self._instrument_coordinator.remove_component(component.name)
            component.close()

        with suppress(Exception):
            self._instrument_coordinator.close()


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

    quantum_device = QuantumDevice(name=name)

    for element_name, element_data in config.items():  # type: str, dict
        element_type = element_data.pop(_DEVICE_ELEMENT_TYPE_PROP, None)
        if not element_type:
            raise ValueError(
                f"Element '{element_name}' is missing a '{_DEVICE_ELEMENT_TYPE_PROP}' specification."
            )

        try:
            module_path, element_cls_name = element_name.rsplit(".", 1)
        except ValueError:
            raise ValueError(
                f"Element '{element_name}' is not a valid full import path for element type."
            )

        module = importlib.import_module(module_path)
        element_class = getattr(module, element_cls_name)

        element_instance = element_class(element_name)
        _apply_parameters(element_instance, element_data)

        quantum_device.add_element(element_instance)

    return quantum_device


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
