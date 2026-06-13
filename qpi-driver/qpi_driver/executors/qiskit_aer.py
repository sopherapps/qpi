from typing import Any

import xarray as xr
from qiskit import transpile

from qpi_driver.compat.qiskit_aer import IS_AER_INSTALLED, AerSimulator
from qpi_driver.executors.base import CircuitPayload, Executor, JobPayload
from qpi_driver.executors.utils.qiskit import load_qasm


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
            payload: JobPayload specifying n_qubits, shots, and circuits.

        Returns:
            xr.Dataset: Dataset containing measured state outcomes, counts, and frequencies.

        Raises:
            ImportError: If qiskit-aer is not installed.
            ValueError: If the provided QASM circuit cannot be loaded.
        """
        n_qubits = payload.n_qubits

        sub_datasets: list[xr.Dataset] = []

        for circ in payload.circuits:
            circ_shots = circ.shots if circ.shots is not None else payload.shots
            qasm_str = circ.circuit
            circuit = load_qasm(qasm_str, n_qubits)

            param_sets = circ.parameter_values or [None]
            for param_vals in param_sets:
                bound_circuit = circuit
                if param_vals is not None and circuit.parameters:
                    bound_circuit = circuit.assign_parameters(param_vals)

                t_qc = transpile(bound_circuit, self._simulator)
                result = self._simulator.run(t_qc, shots=circ_shots, memory=True).result()
                memory = result.get_memory(t_qc)

                ds = self._memory_to_dataset(memory, n_qubits, circ_shots, payload.meas_level)
                sub_datasets.append(ds)

        # Backward compatible: single result → flat dataset (no circuit_index)
        if len(sub_datasets) == 1:
            ds = sub_datasets[0]
            ds.attrs.update({
                "shots": circ_shots,
                "n_qubits": n_qubits,
                "backend": self.name,
                "meas_level": payload.meas_level,
                "meas_return": payload.meas_return,
            })
            return ds

        # Multiple results → concat along circuit_index
        combined = xr.concat(sub_datasets, dim="circuit_index")
        combined.attrs.update({
            "shots": payload.shots,
            "n_qubits": n_qubits,
            "backend": self.name,
            "meas_level": payload.meas_level,
            "meas_return": payload.meas_return,
        })
        return combined

    @staticmethod
    def _memory_to_dataset(
        memory: list[str], n_qubits: int, shots: int, meas_level: int
    ) -> xr.Dataset:
        """Convert Aer shot memory into an xr.Dataset."""
        data_vars = {}
        for i in range(n_qubits):
            # Qubit state 0 or 1 for each shot. Qubit 0 is LSB, so index is n_qubits - 1 - i.
            qubit_vals = [float(outcome[n_qubits - 1 - i]) for outcome in memory]
            data_vars[str(i)] = xr.DataArray(
                [complex(val, 0.0) for val in qubit_vals],
                dims=[f"acq_index_{i}"],
                coords={f"acq_index_{i}": list(range(shots))},
            )
        return xr.Dataset(data_vars)
