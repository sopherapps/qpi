from typing import Any

import xarray as xr
from qiskit import QuantumCircuit, transpile

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
        result = self._simulator.run(t_qc, shots=shots).result()
        counts = result.get_counts(t_qc)

        # Standardise counts keys to binary string representation and pad to 2^n_qubits states
        states_list = [format(i, f"0{n_qubits}b") for i in range(2**n_qubits)]
        counts_list = [counts.get(s, 0) for s in states_list]
        freqs_list = [c / shots for c in counts_list]

        return xr.Dataset(
            {
                "counts": xr.DataArray(
                    counts_list, dims=["state"], coords={"state": states_list}
                ),
                "frequencies": xr.DataArray(
                    freqs_list, dims=["state"], coords={"state": states_list}
                ),
            },
            attrs={"shots": shots, "n_qubits": n_qubits, "backend": self.name},
        )
