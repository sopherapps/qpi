import pytest
import xarray as xr
from qpi_driver.compat.qblox import IS_QBLOX_SCHEDULER_INSTALLED
from qpi_driver.executors import resolve_executor
from qpi_driver.executors.base import CircuitPayload, JobPayload

from .utils import load_json_fixture, load_yaml_fixture

_QUANTIFY_HARDWARE_CONFIG: dict = load_json_fixture("quantify.hardware.json")
_QUANTIFY_DEVICE_CONFIG: dict = load_yaml_fixture("quantify.device.yml")

has_qblox = IS_QBLOX_SCHEDULER_INSTALLED

if has_qblox:
    from qpi_driver.executors.qblox import QbloxExecutor


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
creg c[3];
ccx q[0], q[1], q[2];""",
    # OpenQASM 3.0
    """OPENQASM 3.0;
include "stdgates.inc";
qubit[3] q;
bit[3] c;
ccx q[0], q[1], q[2];""",
]


@pytest.mark.skipif(
    not has_qblox,
    reason="qblox-scheduler and qblox-instruments must be installed to run qblox tests",
)
@pytest.mark.parametrize("qasm", _QASM_PARAMS)
def test_qblox_executor_execute_dummy(qasm):
    """Verify that QbloxExecutor compiles and executes successfully on a dummy cluster with standard output."""
    executor = resolve_executor(
        "qblox",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
    )
    assert isinstance(executor, QbloxExecutor)

    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=100)
    dataset = executor.execute(payload)

    # Assert standardised output format
    assert isinstance(dataset, xr.Dataset)
    assert 0 in dataset or "0" in dataset
    assert 1 in dataset or "1" in dataset
    assert dataset.attrs["shots"] == 100
    assert dataset.attrs["n_qubits"] == 2
    assert dataset.attrs["backend"] == "qblox"

    # In dummy mode without real hardware, the coord length is 1 for each acquisition channel
    assert len(dataset.coords["acq_index_0"]) == 1
    assert len(dataset.coords["acq_index_1"]) == 1


@pytest.mark.skipif(
    not has_qblox,
    reason="qblox-scheduler and qblox-instruments must be installed to run qblox tests",
)
@pytest.mark.parametrize("qasm", _QASM_PARAMS_ONE_QUBIT)
def test_qblox_executor_with_config_fixture(qasm):
    """Verify that QbloxExecutor correctly loads and validates hardware configuration from fixture."""
    executor = resolve_executor(
        "qblox",
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
        is_dummy=True,
    )
    assert isinstance(executor, QbloxExecutor)
    assert executor.hardware_config is not None

    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=50)
    dataset = executor.execute(payload)

    assert isinstance(dataset, xr.Dataset)
    assert 0 in dataset or "0" in dataset
    assert dataset.attrs["shots"] == 50
    assert dataset.attrs["n_qubits"] == 1
    assert len(dataset.coords["acq_index_0"]) == 1


@pytest.mark.skipif(
    not has_qblox,
    reason="qblox-scheduler and qblox-instruments must be installed to run qblox tests",
)
@pytest.mark.parametrize("qasm", _QASM_PARAMS_INVALID)
def test_qblox_executor_invalid_gate_raises(qasm):
    """Verify that invalid gates in QASM raise ValueError."""
    executor = resolve_executor(
        "qblox",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
    )

    # ccx is not a supported gate in QbloxExecutor
    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=10)
    with pytest.raises(ValueError) as excinfo:
        executor.execute(payload)
    assert "not supported" in str(excinfo.value)


@pytest.mark.skipif(
    not has_qblox,
    reason="qblox-scheduler and qblox-instruments must be installed to run qblox tests",
)
def test_qblox_executor_payload_rotation_threshold():
    """Verify that QbloxExecutor respects acq_rotation and acq_threshold in JobPayload."""
    executor = resolve_executor(
        "qblox",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
    )
    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[1];
creg c[1];
x q[0];
measure q[0] -> c[0];"""
    payload = JobPayload(
        circuits=[CircuitPayload(circuit=qasm)],
        shots=10,
        meas_level=2,
        acq_rotation=45.0,
        acq_threshold=0.1,
    )
    dataset = executor.execute(payload)
    assert dataset.attrs.get("acq_rotation") == 45.0
    assert dataset.attrs.get("acq_threshold") == 0.1

    res = executor.process_result(dataset, "job-rotation-threshold")
    assert "counts" in res
