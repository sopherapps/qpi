from contextlib import suppress
from pathlib import Path
from typing import Any

import numpy as np
import xarray as xr

from qpi_driver.compat.qblox import (
    IS_QBLOX_SCHEDULER_INSTALLED,
    HardwareAgent,
    Instrument,
    QbloxHardwareCompilationConfig,
)
from qpi_driver.executors import JobPayload
from qpi_driver.executors.base import Executor
from qpi_driver.executors.qblox.config import (
    load_quantify_hardware_config,
    load_quantum_device,
)
from qpi_driver.executors.qblox.conv import generate_schedule
from qpi_driver.executors.utils.qiskit import load_qasm
from qpi_driver.executors.utils.types import cast_to


class QbloxExecutor(Executor):
    """Executor subclass for interacting with Qblox instruments and modules via qblox-scheduler."""

    def __new__(cls, *args, **kwargs):
        if not IS_QBLOX_SCHEDULER_INSTALLED:
            raise ImportError(
                "qblox-scheduler is not installed. Install the [qblox] extra to use QbloxExecutor."
            )
        return super().__new__(cls)

    def __init__(
        self,
        name: str = "qblox",
        quantify_hardware_config: QbloxHardwareCompilationConfig | Path | dict = Path(
            "quantify.hardware.json"
        ),
        quantify_device_config: Path | dict = Path("quantify.device.json"),
        is_dummy: bool = False,
        data_dir: Path = Path("data"),
        acquisition_timeout: int = 10,
        **kwargs: Any,
    ) -> None:
        """Initialize the QbloxExecutor.

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
        # Clean up any previously registered instruments to avoid name collision errors in QCoDeS
        with suppress(Exception):
            Instrument.close_all()

        self._data_dir = data_dir
        self._is_dummy = is_dummy
        self._acquisition_timeout = acquisition_timeout
        self._hardware_config = load_quantify_hardware_config(quantify_hardware_config)
        self._device = load_quantum_device(name=name, config=quantify_device_config)
        self._agent = HardwareAgent(
            hardware_configuration=self._hardware_config,
            quantum_device_configuration=self._device,
            create_dummy_connections=is_dummy,
            output_dir=data_dir,
        )

    @property
    def hardware_config(self) -> QbloxHardwareCompilationConfig:
        return self._hardware_config

    def execute(self, payload: JobPayload) -> xr.Dataset:
        """Execute quantum instructions using the Qblox scheduler.

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

        schedule = generate_schedule(
            name=payload.id,
            circuit=circuit,
            shots=shots,
            acq_protocol=acq_protocol,
            acq_kwargs=acq_kwargs,
        )

        dataset = self._agent.run(schedule, timeout=self._acquisition_timeout)
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
        """Determine the qblox-scheduler acquisition protocol for the given meas_level.

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
        elements = self._device.elements
        if callable(elements):
            element_names = elements()
        else:
            element_names = list(elements.keys())

        for element_name in element_names:
            if hasattr(self._device, "get_element"):
                element = self._device.get_element(element_name)
            else:
                element = self._device.elements[element_name]
            try:
                threshold = element.measure.acq_threshold
                rotation = element.measure.acq_rotation
                if callable(threshold):
                    threshold = threshold()
                if callable(rotation):
                    rotation = rotation()
                if threshold is not None and rotation is not None:
                    return {
                        "acq_threshold": float(threshold),
                        "acq_rotation": float(rotation),
                    }
            except (AttributeError, KeyError):
                continue
        return {}

    def process_result(self, dataset: xr.Dataset, job_id: str) -> dict:
        """Convert a qblox-scheduler acquisition dataset into a Qiskit-compatible result dict.

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

        meas_level = cast_to(int, dataset.attrs.get("meas_level"), 2)
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
        shots = cast_to(int, dataset.attrs.get("shots"), len(dataset[q0_key]))
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
                r = (
                    float(val.real)
                    if val is not None and not np.isnan(val.real)
                    else 0.0
                )
                i = (
                    float(val.imag)
                    if val is not None and not np.isnan(val.imag)
                    else 0.0
                )
                shot_iq.append([r, i])
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
                    r = val.real if val is not None and not np.isnan(val.real) else 0.0
                    state_chars.append(str(1 if r >= 1 else 0))
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
                    if val is None or (np.isnan(val.real) and np.isnan(val.imag)):
                        state_chars.append("0")
                    else:
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
        with suppress(Exception):
            self._agent.instrument_coordinator.close()
