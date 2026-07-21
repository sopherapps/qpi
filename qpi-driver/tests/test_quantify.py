import importlib.util

import pytest
import xarray as xr
from qiskit import QuantumCircuit
from qpi_driver.executors import resolve_executor
from qpi_driver.executors.base import CircuitPayload, JobPayload

from .utils import (
    equal_up_to_global_phase,
    load_json_fixture,
    load_yaml_fixture,
    operations_to_unitary,
    toffoli_unitary,
)

_QUANTIFY_HARDWARE_CONFIG: dict = load_json_fixture("quantify.hardware.json")
_QUANTIFY_DEVICE_CONFIG: dict = load_yaml_fixture("quantify.device.yml")

has_quantify = (
    importlib.util.find_spec("quantify_scheduler") is not None
    and importlib.util.find_spec("qblox_instruments") is not None
)

pytestmark = pytest.mark.skipif(
    not has_quantify,
    reason="quantify-scheduler and qblox-instruments must be installed to run quantify tests",
)

if has_quantify:
    from qpi_driver.executors.quantify import QuantifyExecutor
    from qpi_driver.executors.quantify.conv import to_quantify_gates


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
qreg q[2];
cy q[0], q[1];""",
    # OpenQASM 3.0
    """OPENQASM 3.0;
include "stdgates.inc";
qubit[2] q;
cy q[0], q[1];""",
]


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

    # acq_index_N is the acquisition-index axis (one measurement per qubit -> length 1),
    # not the shot count; single-shot data lives on a separate "repetition" dimension.
    assert len(dataset.coords["acq_index_0"]) == 1
    assert len(dataset.coords["acq_index_1"]) == 1

    # meas_level=2 must discriminate every shot (BinMode.APPEND), so the counts
    # add up to the shot count rather than collapsing to a single sample.
    result = executor.process_result(dataset, "job-execute-dummy")
    counts = result["counts"]
    assert sum(counts.values()) == 100
    assert len(counts) == 2 ** dataset.attrs["n_qubits"]


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


def test_quantify_executor_payload_rotation_threshold():
    """Verify that QuantifyExecutor respects acq_rotation and acq_threshold in JobPayload."""
    executor = resolve_executor(
        "quantify",
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


def test_quantify_executor_flux_tunable_coupler():
    """Verify that QuantifyExecutor compiles a CZ gate on qubits with a flux tunable coupler, overriding the port."""
    executor = resolve_executor(
        "quantify",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
    )
    assert isinstance(executor, QuantifyExecutor)

    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[3];
creg c[3];
cz q[1], q[2];
measure q[1] -> c[1];
measure q[2] -> c[2];"""

    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=10)
    dataset = executor.execute(payload)
    assert dataset is not None

    device_config = executor._device.generate_device_config()
    edge_config = device_config.edges["q1_q2"]
    cz_config = edge_config["CZ"]
    assert cz_config.factory_kwargs["square_port"] == "q1_q2:fl"
    assert cz_config.factory_kwargs["square_clock"] == "q1_q2.cz"


def test_quantify_executor_repeated_measurement():
    """A qubit measured more than once yields acq_index length > 1 per shot.

    Running it end-to-end guards the discrimination against raising on that
    ragged per-shot array, confirms every shot is still counted, and -- since
    the qubit is measured into two distinct clbits (c[0] and c[1]) -- the
    counts bitstring must be 2 bits wide rather than collapsing to a single
    bit representing only the qubit's final measurement.
    """
    executor = resolve_executor(
        "quantify",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
    )

    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[1];
creg c[2];
x q[0];
measure q[0] -> c[0];
measure q[0] -> c[1];"""

    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=50)
    dataset = executor.execute(payload)

    result = executor.process_result(dataset, "job-repeated-measurement")
    counts = result["counts"]
    assert sum(counts.values()) == 50
    assert len(counts) == 4
    assert all(len(state) == 2 for state in counts)


def test_quantify_executor_partial_measurement_pads_unmeasured_clbit():
    """Measuring only c[1] of a 2-bit creg must yield a 2-bit string with c[0] as a fixed '0' gap."""
    executor = resolve_executor(
        "quantify",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
    )

    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
x q[1];
measure q[1] -> c[1];"""

    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=20)
    dataset = executor.execute(payload)

    result = executor.process_result(dataset, "job-partial-measurement")
    counts = result["counts"]
    assert sum(counts.values()) == 20
    assert len(counts) == 4
    # c[0] was never measured, so it must always stay "0" (the rightmost bit)
    # in every state that actually occurred (zero-padded states are excluded,
    # since padding fills in all 2**num_clbits keys regardless of c[0]).
    assert all(state[-1] == "0" for state, n in counts.items() if n > 0)


def test_quantify_executor_clbit_remap():
    """measure q[0]->c[1]; q[1]->c[0] must still yield a well-formed 2-bit-wide counts dict.

    The dummy cluster fakes acquisition data rather than simulating true
    qubit state, so exact bit values aren't asserted here; the shared unit
    tests in tests/test_counts.py cover exact bit placement.
    """
    executor = resolve_executor(
        "quantify",
        is_dummy=True,
        quantify_hardware_config=_QUANTIFY_HARDWARE_CONFIG,
        quantify_device_config=_QUANTIFY_DEVICE_CONFIG,
    )

    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
measure q[0] -> c[1];
measure q[1] -> c[0];"""

    payload = JobPayload(circuits=[CircuitPayload(circuit=qasm)], shots=20)
    dataset = executor.execute(payload)

    result = executor.process_result(dataset, "job-clbit-remap")
    counts = result["counts"]
    assert sum(counts.values()) == 20
    assert len(counts) == 4


def test_quantify_ccx_decomposes_to_toffoli():
    """The CCXGate conversion must implement a real Toffoli, not a bare CX(c1, t)."""
    circuit = QuantumCircuit(3)
    circuit.ccx(0, 1, 2)
    instruction = circuit.data[0]

    operations = to_quantify_gates(
        circuit=circuit, instruction=instruction, acq_indices={}
    )

    unitary = operations_to_unitary(operations, ["q0", "q1", "q2"])
    assert equal_up_to_global_phase(unitary, toffoli_unitary())
