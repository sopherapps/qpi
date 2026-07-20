from typing import Any

import xarray as xr
from qiskit import transpile

from qpi_driver.compat.qiskit_aer import IS_AER_INSTALLED, AerSimulator
from qpi_driver.executors import JobPayload
from qpi_driver.executors.base import Executor
from qpi_driver.executors.utils.qiskit import load_qasm, memory_to_dataset
from qpi_driver.executors.utils.result import build_qiskit_result, iq_memory_avg
from qpi_driver.executors.utils.types import cast_to


class QiskitAerExecutor(Executor):
    def __init__(self, name: str = "qiskit_aer", **kwargs: Any):
        if not IS_AER_INSTALLED:
            raise ImportError(
                "qiskit-aer is not installed. Install the [aer] extra to use QiskitAerExecutor."
            )

        super().__init__(name, **kwargs)
        self._simulator = AerSimulator()

    def execute(self, payload: JobPayload) -> xr.Dataset:
        """Run quantum circuit simulation using Qiskit Aer backend.

        For multi-circuit payloads, results are concatenated along a ``circuit_index``
        dimension.  A single-circuit payload without parameter bindings returns the
        legacy flat format (no ``circuit_index`` dimension) for backward compatibility.

        Args:
            payload: JobPayload specifying shots and circuits.

        Returns:
            xr.Dataset: Dataset containing measured state outcomes, counts, and frequencies.

        Raises:
            ImportError: If qiskit-aer is not installed.
            ValueError: If the provided QASM circuit cannot be loaded.
        """
        sub_datasets: list[xr.Dataset] = []

        for circ in payload.circuits:
            circ_shots = circ.shots if circ.shots is not None else payload.shots
            qasm_str = circ.circuit
            circuit = load_qasm(qasm_str)

            param_sets = circ.parameter_values or [None]
            for param_vals in param_sets:
                bound_circuit = circuit
                if param_vals is not None and circuit.parameters:
                    bound_circuit = circuit.assign_parameters(param_vals)

                t_qc = transpile(bound_circuit, self._simulator)
                result = self._simulator.run(
                    t_qc, shots=circ_shots, memory=True
                ).result()
                memory = result.get_memory(t_qc)
                n_qubits = circuit.num_qubits

                ds = memory_to_dataset(memory, n_qubits, circ_shots, payload.meas_level)
                sub_datasets.append(ds)

        # Backward compatible: single result → flat dataset (no circuit_index)
        if len(sub_datasets) == 1:
            ds = sub_datasets[0]
            ds.attrs.update(
                {
                    "shots": circ_shots,
                    "n_qubits": n_qubits,
                    "backend": self.name,
                    "meas_level": payload.meas_level,
                    "meas_return": payload.meas_return,
                }
            )
            return ds

        # Multiple results → concat along circuit_index
        combined = xr.concat(sub_datasets, dim="circuit_index")
        combined.attrs.update(
            {
                "shots": payload.shots,
                "n_qubits": n_qubits,
                "backend": self.name,
                "meas_level": payload.meas_level,
                "meas_return": payload.meas_return,
            }
        )
        return combined

    def process_result(self, dataset: xr.Dataset, job_id: str) -> dict:
        """Convert the simulator's xr.Dataset into a Qiskit-compatible result dict.

        Supports meas_level 1 (IQ memory) and 2 (classified counts).
        meas_level 0 is not supported for simulators.

        Args:
            dataset: xr.Dataset from execute().
            job_id: Unique job ID.

        Returns:
            dict: Qiskit-compatible result dict.
        """
        meas_level = cast_to(int, dataset.attrs.get("meas_level"), 2)
        meas_return = str(dataset.attrs.get("meas_return", "single"))
        backend = dataset.attrs.get("backend", self.name)

        # Handle multi-circuit datasets
        if "circuit_index" in dataset.dims:
            circuit_results = []
            for ci in range(dataset.sizes["circuit_index"]):
                sub_ds = dataset.isel(circuit_index=ci)
                sub_ds.attrs.update(dataset.attrs)
                circuit_results.append(
                    self._single_dataset_to_result(sub_ds, meas_level, meas_return)
                )
            return build_qiskit_result(circuit_results, job_id, backend)

        # Single circuit
        single = self._single_dataset_to_result(dataset, meas_level, meas_return)
        return build_qiskit_result([single], job_id, backend)

    @staticmethod
    def _single_dataset_to_result(
        dataset: xr.Dataset, meas_level: int, meas_return: str
    ) -> dict:
        """Extract counts or IQ memory from a single-circuit dataset slice."""
        # Identify qubit variables (integer-named data vars)
        qubit_vars = []
        for var in dataset.data_vars:
            try:
                qubit_vars.append(int(var))
            except ValueError:
                pass
        if not qubit_vars:
            return {"raw": str(dataset), "shots": 0}

        qubit_vars.sort()
        q0_key = str(qubit_vars[0])
        shots = cast_to(int, dataset.attrs.get("shots"), len(dataset[q0_key]))

        # meas_level=1: return IQ memory
        if meas_level == 1:
            memory: list[list[list[float]]] = []
            num_samples = len(dataset[q0_key])
            for s in range(num_samples):
                shot_iq = []
                for q_idx in qubit_vars:
                    val = dataset[str(q_idx)].values[s]
                    shot_iq.append([float(val.real), float(val.imag)])
                memory.append(shot_iq)

            if meas_return == "avg" and memory:
                memory = iq_memory_avg(memory, len(qubit_vars))

            return {"memory": memory, "shots": shots}

        # meas_level=2: classified counts
        n_qubits = len(qubit_vars)
        counts_dict: dict[str, int] = {}
        num_samples = len(dataset[q0_key])
        for s in range(num_samples):
            state_chars = []
            for q_idx in reversed(qubit_vars):
                val = dataset[str(q_idx)].values[s]
                state_chars.append("1" if val.real > 0.5 else "0")
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
