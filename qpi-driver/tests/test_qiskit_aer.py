import importlib.util

import pytest
import xarray as xr
from qpi_driver.executors import resolve_executor
from qpi_driver.executors.base import CircuitPayload, JobPayload
from qpi_driver.executors.qiskit_aer import QiskitAerExecutor

has_aer = importlib.util.find_spec("qiskit_aer") is not None


_QASM_PARAMS = [
    # OpenQASM 2.0
    """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
h q[0];
cx q[0], q[1];
measure q -> c;""",
    # OpenQASM 3.0
    """OPENQASM 3.0;
include "stdgates.inc";
qubit[2] q;
bit[2] c;
h q[0];
cx q[0], q[1];
c[0] = measure q[0];
c[1] = measure q[1];""",
]


@pytest.mark.skipif(
    not has_aer, reason="qiskit-aer must be installed to run qiskit-aer tests"
)
@pytest.mark.parametrize("qasm", _QASM_PARAMS)
def test_qiskit_aer_executor_execute(qasm):
    """Verify that QiskitAerExecutor runs successfully and returns correct standard output."""
    executor = resolve_executor("qiskit_aer")
    assert isinstance(executor, QiskitAerExecutor)

    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=100, n_qubits=2)
    dataset = executor.execute(payload)

    assert isinstance(dataset, xr.Dataset)
    assert "0" in dataset
    assert "1" in dataset
    assert len(dataset.coords["acq_index_0"]) == 100
    assert len(dataset.coords["acq_index_1"]) == 100
    assert dataset.attrs["shots"] == 100
    assert dataset.attrs["n_qubits"] == 2
    assert dataset.attrs["backend"] == "qiskit_aer"
