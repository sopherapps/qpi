import importlib.util

import pytest
import xarray as xr
from qpi_driver.executors import resolve_executor
from qpi_driver.executors.base import CircuitPayload, JobPayload
from utils import load_json_fixture, load_yaml_fixture

_QUANTIFY_HARDWARE_CONFIG: dict = load_json_fixture("quantify.hardware.json")
_QUANTIFY_DEVICE_CONFIG: dict = load_yaml_fixture("quantify.device.yml")

has_quantify = (
    importlib.util.find_spec("quantify_scheduler") is not None
    and importlib.util.find_spec("qblox_instruments") is not None
)

if has_quantify:
    from qpi_driver.executors.quantify import QuantifyExecutor


_QASM_PARAMS = [
    # OpenQASM 2.0
    """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
x q[0];
measure q[0] -> c[0];
measure q[1] -> c[1];""",
    # OpenQASM 3.0
    """OPENQASM 3.0;
include "stdgates.inc";
qubit[2] q;
bit[2] c;
x q[0];
c[0] = measure q[0];
c[1] = measure q[1];""",
]


_QASM_PARAMS_ONE_QUBIT = [
    # OpenQASM 2.0
    """OPENQASM 2.0;
include "qelib1.inc";
qreg q[1];
creg c[1];
x q[0];
measure q[0] -> c[0];""",
    # OpenQASM 3.0
    """OPENQASM 3.0;
include "stdgates.inc";
qubit[1] q;
bit[1] c;
x q[0];
c[0] = measure q[0];""",
]

_QASM_PARAMS_INVALID = [
    # OpenQASM 2.0
    """OPENQASM 2.0;
include "qelib1.inc";
qreg q[3];
ccx q[0], q[1], q[2];""",
    # OpenQASM 3.0
    """OPENQASM 3.0;
include "stdgates.inc";
qubit[3] q;
ccx q[0], q[1], q[2];""",
]


@pytest.mark.skipif(
    not has_quantify,
    reason="quantify-scheduler and qblox-instruments must be installed to run quantify tests",
)
@pytest.mark.parametrize("qasm", _QASM_PARAMS)
def test_quantify_executor_execute_dummy(qasm):
    """Verify that QuantifyExecutor compiles and executes successfully on a dummy cluster with standard output."""
    executor = resolve_executor(
        "quantify",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
    )
    assert isinstance(executor, QuantifyExecutor)

    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=100)
    dataset = executor.execute(payload)

    # Assert standardised output format
    assert isinstance(dataset, xr.Dataset)
    assert 0 in dataset or "0" in dataset
    assert 1 in dataset or "1" in dataset
    assert dataset.attrs["shots"] == 100
    assert dataset.attrs["n_qubits"] == 2
    assert dataset.attrs["backend"] == "quantify"

    # In dummy mode without real hardware, the coord length is 1 for each acquisition channel
    assert len(dataset.coords["acq_index_0"]) == 1
    assert len(dataset.coords["acq_index_1"]) == 1


@pytest.mark.skipif(
    not has_quantify,
    reason="quantify-scheduler and qblox-instruments must be installed to run quantify tests",
)
@pytest.mark.parametrize("qasm", _QASM_PARAMS_ONE_QUBIT)
def test_quantify_executor_with_config_fixture(qasm):
    """Verify that QuantifyExecutor correctly loads and validates hardware configuration from fixture."""
    executor = resolve_executor(
        "quantify",
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
        is_dummy=True,
    )
    assert isinstance(executor, QuantifyExecutor)
    assert executor.hardware_config is not None

    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=50)
    dataset = executor.execute(payload)

    assert isinstance(dataset, xr.Dataset)
    assert 0 in dataset or "0" in dataset
    assert dataset.attrs["shots"] == 50
    assert dataset.attrs["n_qubits"] == 1
    assert len(dataset.coords["acq_index_0"]) == 1


@pytest.mark.skipif(
    not has_quantify,
    reason="quantify-scheduler and qblox-instruments must be installed to run quantify tests",
)
@pytest.mark.parametrize("qasm", _QASM_PARAMS_INVALID)
def test_quantify_executor_invalid_gate_raises(qasm):
    """Verify that invalid gates in QASM raise ValueError."""
    executor = resolve_executor(
        "quantify",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
    )

    # ccx is not a supported gate in QuantifyExecutor
    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=10)
    with pytest.raises(ValueError) as excinfo:
        executor.execute(payload)
    assert "not supported" in str(excinfo.value)
