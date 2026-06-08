import xarray as xr

from qpi_driver.executors.base import Executor, JobPayload


class QiskitAerExecutor(Executor):
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
        try:
            from qiskit import QuantumCircuit, transpile
            from qiskit_aer import AerSimulator
        except ImportError as exc:
            raise ImportError(
                "qiskit-aer is not installed. Install the [aer] extra to use QiskitAerExecutor."
            ) from exc

        n_qubits = payload.n_qubits
        shots = payload.shots
        qasm_str = payload.qasm

        if qasm_str is None or qasm_str == "":
            raise ValueError("No circuit provided in payload")
        if (
            qasm_str
            and isinstance(qasm_str, str)
            and ("OPENQASM" in qasm_str or "qreg" in qasm_str or "include" in qasm_str)
        ):
            try:
                try:
                    import qiskit.qasm3 as qasm3

                    qc = qasm3.loads(qasm_str)
                except Exception:
                    qc = QuantumCircuit.from_qasm_str(qasm_str)
            except Exception as exc:
                raise ValueError(f"Failed to parse QASM circuit: {exc}") from exc
        else:
            raise ValueError("Invalid circuit provided in payload")

        simulator = AerSimulator()
        t_qc = transpile(qc, simulator)
        sim_result = simulator.run(t_qc, shots=shots).result()
        counts = sim_result.get_counts(t_qc)

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
            attrs={"shots": shots, "n_qubits": n_qubits, "backend": "qiskit_aer"},
        )
