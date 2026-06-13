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

    customs: dict[str, type[Executor]] = {"dummy": DummyExecutor}
    exec_inst = resolve_executor("dummy", custom_executors=customs)
    assert isinstance(exec_inst, DummyExecutor)


def test_placeholder_executors_raise_not_implemented():
    """Verify that placeholder executors raise NotImplementedError."""
    from qpi_driver.executors.base import JobPayload

    payload = JobPayload(circuits=[CircuitPayload(circuit="OPENQASM 2.0;")])
    for name in ["qblox", "presto"]:
        executor = resolve_executor(name)
        with pytest.raises(NotImplementedError):
            executor.execute(payload)
