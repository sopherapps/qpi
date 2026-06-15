import pytest
import xarray as xr
from qpi_driver.executors import resolve_executor
from qpi_driver.executors.base import CircuitPayload, Executor
from qpi_driver.executors.mock import MockExecutor


def test_mock_executor_execute():
    """Verify that MockExecutor runs successfully and returns the correct xarray.Dataset format."""
    executor = MockExecutor(name="mock")
    from qpi_driver.executors.base import JobPayload

    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
h q[0];
cx q[0], q[1];
measure q -> c;"""
    payload_dict = {"shots": 500, "circuit": qasm}
    payload = JobPayload.from_dict(payload_dict)
    dataset = executor.execute(payload)

    # Assert return type
    assert isinstance(dataset, xr.Dataset)

    # Assert data variables and coordinates
    assert "0" in dataset
    assert "1" in dataset
    assert len(dataset.coords["acq_index_0"]) == 500
    assert len(dataset.coords["acq_index_1"]) == 500

    # Assert attributes
    assert dataset.attrs.get("shots") == 500
    assert dataset.attrs.get("backend") == "mock"


def test_resolve_executor():
    """Verify executor resolution logic for strings, classes, and instances."""
    # Resolve by string name
    exec_inst = resolve_executor("mock")
    assert isinstance(exec_inst, MockExecutor)

    # Resolve by class
    exec_inst = resolve_executor(MockExecutor)
    assert isinstance(exec_inst, MockExecutor)

    # Resolve by instance
    inst = MockExecutor()
    exec_inst = resolve_executor(inst)
    assert exec_inst is inst

    # Handle resolution error for unknown name
    with pytest.raises(ValueError) as excinfo:
        resolve_executor("non_existent_executor")
    assert "Unknown executor name" in str(excinfo.value)

    # Handle resolution error for invalid type
    with pytest.raises(TypeError):
        resolve_executor(123)


def test_custom_executor_resolution():
    """Verify resolver successfully supports custom user-defined executors."""

    from qpi_driver.executors.base import JobPayload

    class DummyExecutor(Executor):
        def execute(self, payload: JobPayload) -> xr.Dataset:
            return xr.Dataset()

        def process_result(self, dataset: xr.Dataset, job_id: str) -> dict:
            return {}

    customs: dict[str, type[Executor]] = {"dummy": DummyExecutor}
    exec_inst = resolve_executor("dummy", custom_executors=customs)
    assert isinstance(exec_inst, DummyExecutor)


def test_placeholder_executors_raise_not_implemented():
    """Verify that placeholder executors raise NotImplementedError."""
    from qpi_driver.executors.base import JobPayload

    payload = JobPayload(circuits=[CircuitPayload(circuit="OPENQASM 2.0;")])
    for name in ["presto"]:
        executor = resolve_executor(name)
        with pytest.raises(NotImplementedError):
            executor.execute(payload)


def test_mock_executor_process_result():
    """Verify that MockExecutor.process_result converts xarray.Dataset into correct Qiskit-compatible results."""
    from qpi_driver.executors.base import JobPayload

    executor = MockExecutor(name="mock")

    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
x q[0];
measure q -> c;"""

    # Test meas_level=2 (classified counts)
    payload_counts = JobPayload(
        circuits=[CircuitPayload(circuit=qasm)], meas_level=2, shots=100
    )
    dataset_counts = executor.execute(payload_counts)
    res_counts = executor.process_result(dataset_counts, "job-counts")

    assert res_counts["backend"] == "mock"
    assert res_counts["shots"] == 100
    assert "counts" in res_counts
    assert "01" in res_counts["counts"]

    # Test meas_level=1 (IQ memory)
    payload_iq = JobPayload(
        circuits=[CircuitPayload(circuit=qasm)], meas_level=1, shots=10
    )
    dataset_iq = executor.execute(payload_iq)
    res_iq = executor.process_result(dataset_iq, "job-iq")

    assert res_iq["backend"] == "mock"
    assert "memory" in res_iq
    assert len(res_iq["memory"]) == 10  # shots
    assert len(res_iq["memory"][0]) == 2  # qubits
    assert len(res_iq["memory"][0][0]) == 2  # [real, imag]
