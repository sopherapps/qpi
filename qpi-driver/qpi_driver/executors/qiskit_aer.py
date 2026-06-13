from typing import Any

import xarray as xr
from qiskit import transpile

from qpi_driver.compat.qiskit_aer import IS_AER_INSTALLED, AerSimulator
from qpi_driver.executors.base import Executor, JobPayload
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

        Args:
            payload: JobPayload specifying n_qubits, shots, and qasm.

        Returns:
            xr.Dataset: Dataset containing measured state outcomes, counts, and frequencies.

        Raises:
            ImportError: If qiskit-aer is not installed.
            ValueError: If the provided QASM circuit cannot be loaded.
        """
        n_qubits = payload.n_qubits
        shots = payload.shots
        qasm_str = payload.qasm
        circuit = load_qasm(qasm_str, n_qubits)

        t_qc = transpile(circuit, self._simulator)
        result = self._simulator.run(t_qc, shots=shots, memory=True).result()
        memory = result.get_memory(t_qc)

        data_vars = {}
        for i in range(n_qubits):
            # Qubit state 0 or 1 for each shot. Qubit 0 is LSB, so index is n_qubits - 1 - i.
            qubit_vals = [float(outcome[n_qubits - 1 - i]) for outcome in memory]
            data_vars[str(i)] = xr.DataArray(
                [complex(val, 0.0) for val in qubit_vals],
                dims=[f"acq_index_{i}"],
                coords={f"acq_index_{i}": list(range(shots))},
            )

        return xr.Dataset(
            data_vars,
            attrs={"shots": shots, "n_qubits": n_qubits, "backend": self.name},
        )
