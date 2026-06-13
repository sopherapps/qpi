from qiskit import QuantumCircuit, qasm2, qasm3


def load_qasm(value: str, num_qubits: int | None = None) -> QuantumCircuit:
    """Loads QASM from the QASM string

    Args:
        value: the QASM string from which to load the circuit
        num_qubits: provides number of physical/virtual qubits

    Returns:
        the QASM circuit as got from the string value passed

    Raises:
        ValueError: Failed to parse QASM circuit: exc
    """
    try:
        try:
            return qasm3.loads(value, num_qubits=num_qubits)
        except qasm3.QASM3Error:
            return qasm2.loads(value, strict=True)
    except Exception as exc:
        raise ValueError(f"Failed to parse QASM circuit: {exc}") from exc
