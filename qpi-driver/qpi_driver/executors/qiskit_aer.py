import xarray as xr

from qpi_driver.executors.base import Executor


class QiskitAerExecutor(Executor):
    def execute(self, payload: dict) -> xr.Dataset:
        """Run quantum circuit simulation using Qiskit Aer backend.

        Args:
            payload: Dictionary specifying 'n_qubits', 'shots', and optional QASM string ('circuit_qasm' or 'circuit').

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

        n_qubits = payload.get("n_qubits", 2)
        shots = payload.get("shots", 1024)

        # Support dynamically passing the circuit in the payload as a QASM string.
        # If not provided or if it's just a simple name, fall back to the default Bell state.
        qasm_str = payload.get("circuit_qasm") or payload.get("circuit")
        if (
            qasm_str
            and isinstance(qasm_str, str)
            and ("OPENQASM" in qasm_str or "qreg" in qasm_str or "include" in qasm_str)
        ):
            try:
                if "OPENQASM 3" in qasm_str or "OPENQASM 3.0" in qasm_str:
                    import qiskit.qasm3 as qasm3

                    qc = qasm3.loads(qasm_str)
                else:
                    qc = QuantumCircuit.from_qasm_str(qasm_str)
            except Exception as exc:
                raise ValueError(f"Failed to parse QASM circuit: {exc}") from exc
        else:
            qc = QuantumCircuit(n_qubits)
            qc.h(0)
            if n_qubits > 1:
                qc.cx(0, 1)
            qc.measure_all()

        simulator = AerSimulator()
        t_qc = transpile(qc, simulator)
        sim_result = simulator.run(t_qc, shots=shots).result()
        counts = sim_result.get_counts(t_qc)

        # Standardise counts keys to binary string representation
        # Qiskit get_counts keys are strings like '00', '01'
        states = list(counts.keys())
        freqs = [counts[s] / shots for s in states]

        return xr.Dataset(
            {
                "counts": xr.DataArray(
                    list(counts.values()), dims=["state"], coords={"state": states}
                ),
                "frequencies": xr.DataArray(
                    freqs, dims=["state"], coords={"state": states}
                ),
            },
            attrs={"shots": shots, "n_qubits": n_qubits, "backend": "qiskit_aer"},
        )
