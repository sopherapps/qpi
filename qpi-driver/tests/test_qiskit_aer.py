import pytest
import xarray as xr
from qpi_driver.executors import resolve_executor
from qpi_driver.executors.qiskit_aer import QiskitAerExecutor
from qpi_driver.executors.base import JobPayload

try:
    import qiskit_aer
    has_aer = True
except ImportError:
    has_aer = False

@pytest.mark.skipif(
    not has_aer,
    reason="qiskit-aer must be installed to run qiskit-aer tests"
)
def test_qiskit_aer_executor_execute():
    """Verify that QiskitAerExecutor runs successfully and returns correct standard output."""
    executor = resolve_executor("qiskit_aer")
    assert isinstance(executor, QiskitAerExecutor)

    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
h q[0];
cx q[0], q[1];
measure q -> c;"""

    payload = JobPayload(qasm=qasm, shots=100, n_qubits=2)
    dataset = executor.execute(payload)

    assert isinstance(dataset, xr.Dataset)
    assert "counts" in dataset
    assert "frequencies" in dataset
    assert dataset.attrs["shots"] == 100
    assert dataset.attrs["n_qubits"] == 2
    assert dataset.attrs["backend"] == "qiskit_aer"

    counts_da = dataset["counts"]
    states = counts_da.coords["state"].values.tolist()
    counts = counts_da.values.tolist()

    assert len(states) == 4
    assert sum(counts) == 100
