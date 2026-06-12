import importlib.util

import pytest
import xarray as xr
from qpi_driver.executors import resolve_executor
from qpi_driver.executors.base import JobPayload
from qpi_driver.executors.quantify import QuantifyExecutor
from utils import load_json_fixture

_QUANTIFY_HARDWARE_CONFIG: dict = load_json_fixture("quantify.hardware.json")

has_quantify = (
    importlib.util.find_spec("quantify_scheduler") is not None
    and importlib.util.find_spec("qblox_instruments") is not None
)


@pytest.mark.skipif(
    not has_quantify,
    reason="quantify-scheduler and qblox-instruments must be installed to run quantify tests",
)
def test_quantify_executor_execute_dummy():
    """Verify that QuantifyExecutor compiles and executes successfully on a dummy cluster with standard output."""
    executor = resolve_executor(
        "quantify", is_dummy=True, quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG
    )
    assert isinstance(executor, QuantifyExecutor)

    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
x q[0];
measure q[0] -> c[0];
measure q[1] -> c[1];"""

    payload = JobPayload(qasm=qasm, shots=100, n_qubits=2)
    dataset = executor.execute(payload)

    # Assert standardised output format
    assert isinstance(dataset, xr.Dataset)
    assert "counts" in dataset
    assert "frequencies" in dataset
    assert dataset.attrs["shots"] == 100
    assert dataset.attrs["n_qubits"] == 2
    assert dataset.attrs["backend"] == "quantify"

    counts_da = dataset["counts"]
    assert "state" in counts_da.coords
    states = counts_da.coords["state"].values.tolist()
    counts = counts_da.values.tolist()

    assert len(states) == 4
    assert sum(counts) == 100


def test_quantify_executor_with_config_fixture():
    """Verify that QuantifyExecutor correctly loads and validates hardware configuration from fixture."""
    executor = resolve_executor(
        "quantify", quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG, is_dummy=True
    )
    assert isinstance(executor, QuantifyExecutor)
    assert executor.hardware_config is not None

    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[1];
creg c[1];
x q[0];
measure q[0] -> c[0];"""

    payload = JobPayload(qasm=qasm, shots=50, n_qubits=1)
    dataset = executor.execute(payload)

    assert isinstance(dataset, xr.Dataset)
    assert "counts" in dataset
    assert dataset.attrs["shots"] == 50
    assert dataset.attrs["n_qubits"] == 1


def test_quantify_executor_invalid_gate_raises():
    """Verify that invalid gates in QASM raise ValueError."""
    executor = resolve_executor(
        "quantify",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
    )

    # ccx is not a supported gate in QuantifyExecutor
    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[3];
ccx q[0], q[1], q[2];"""

    payload = JobPayload(qasm=qasm, shots=10, n_qubits=3)
    with pytest.raises(ValueError) as excinfo:
        executor.execute(payload)
    assert "not supported" in str(excinfo.value)
