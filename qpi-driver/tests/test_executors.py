import pytest
import xarray as xr
from qpi_driver.executors import resolve_executor
from qpi_driver.executors.base import Executor
from qpi_driver.executors.mock import MockExecutor


def test_mock_executor_execute():
    """Verify that MockExecutor runs successfully and returns the correct xarray.Dataset format."""
    executor = MockExecutor()
    from qpi_driver.executors.base import JobPayload

    payload_dict = {"n_qubits": 2, "shots": 500, "circuit": "bell_state"}
    payload = JobPayload.from_dict(payload_dict)
    dataset = executor.execute(payload)

    # Assert return type
    assert isinstance(dataset, xr.Dataset)

    # Assert data variables and coordinates
    assert "counts" in dataset
    counts_da = dataset["counts"]
    assert "state" in counts_da.coords

    # Assert attributes
    assert dataset.attrs.get("shots") == 500
    assert dataset.attrs.get("backend") == "mock"

    # Check count values sum to total shots
    states = counts_da.coords["state"].values.tolist()
    counts = counts_da.values.tolist()
    assert len(states) == 4  # 2^2 states: '00', '01', '10', '11'
    assert sum(counts) == 500


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

    payload = JobPayload(qasm="OPENQASM 2.0;")
    for name in ["qblox", "presto"]:
        executor = resolve_executor(name)
        with pytest.raises(NotImplementedError):
            executor.execute(payload)
