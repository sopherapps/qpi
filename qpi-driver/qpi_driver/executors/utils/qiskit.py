from qiskit import QuantumCircuit, qasm3


def load_qasm(value: str) -> QuantumCircuit:
    """Loads QASM from the QASM string

    Args:
        value: the QASM string from which to load the circuit

    Returns:
        the QASM circuit as got from the string value passed

    Raises:
        ValueError: Failed to parse QASM circuit: exc
    """
    try:
        try:
            return qasm3.loads(value)
        except Exception:
            return QuantumCircuit.from_qasm_str(value)
    except Exception as exc:
        raise ValueError(f"Failed to parse QASM circuit: {exc}") from exc
