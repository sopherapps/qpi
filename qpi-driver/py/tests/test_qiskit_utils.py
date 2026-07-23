import pytest
from qiskit import QuantumCircuit
from qpi_driver.executors.utils.qiskit import load_qasm

_QASM2_CIRCUIT = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
h q[0];
cx q[0], q[1];
measure q -> c;"""

_QASM3_CIRCUIT = """OPENQASM 3.0;
include "stdgates.inc";
qubit[2] q;
bit[2] c;
h q[0];
cx q[0], q[1];
c = measure q;"""

_QASM3_OUT_OF_RANGE_QUBIT_CIRCUIT = """OPENQASM 3.0;
include "stdgates.inc";
qubit[2] q;
bit[2] c;
h q[11];
cx q[11], q[12];
c = measure q;"""


def test_load_qasm_openqasm2():
    circuit = load_qasm(_QASM2_CIRCUIT)

    assert isinstance(circuit, QuantumCircuit)
    assert circuit.num_qubits == 2


def test_load_qasm_openqasm3():
    circuit = load_qasm(_QASM3_CIRCUIT)

    assert isinstance(circuit, QuantumCircuit)
    assert circuit.num_qubits == 2


def test_load_qasm_missing_version_header():
    with pytest.raises(ValueError, match="missing or unrecognized OPENQASM"):
        load_qasm("qreg q[2];\ncreg c[2];\nh q[0];\nmeasure q -> c;")


def test_load_qasm_openqasm3_error_is_not_masked_by_openqasm2_fallback():
    """A genuine OpenQASM 3 error (out-of-range qubit index) must surface as-is,
    not get overwritten by an unrelated OpenQASM 2 parse error."""
    with pytest.raises(ValueError) as exc_info:
        load_qasm(_QASM3_OUT_OF_RANGE_QUBIT_CIRCUIT)

    message = str(exc_info.value)
    assert "OpenQASM 2.0" not in message
    assert "out of range" in message
