from typing import Literal, Optional

import numpy as np
import qiskit.circuit
from qiskit import QuantumCircuit
from qiskit.circuit import library as qiskit_library

from qpi_driver.compat.qblox import (
    CZ,
    H,
    IdlePulse,
    Measure,
    Operation,
    Reset,
    Rxy,
    Rz,
    S,
    Schedule,
    SDagger,
    T,
    TDagger,
    X,
    Y,
    Z,
)

_RefPtNewType = Optional[Literal["start", "center", "end"]]


def to_qblox_gates(
    circuit: QuantumCircuit,
    instruction: qiskit.circuit.CircuitInstruction,
    acq_indices: dict[int, int],
    acq_protocol: str = "SSBIntegrationComplex",
    acq_kwargs: dict | None = None,
) -> list[Operation]:
    """Converts a qiskit Instruction to Qblox gate operations.

    Args:
        circuit: The circuit in which the instruction is.
        instruction: The instruction to convert.
        acq_indices: A mapping of qubit and current acquisitions/measurements done on said qubit in circuit..

    Returns:
        list of qblox Operations

    Raises:
        ValueError: if the gate is not supported by QbloxExecutor
    """
    gate = instruction.operation
    name = instruction.name
    params = instruction.params

    qubit_indices = [circuit.find_bit(q).index for q in instruction.qubits]
    qubits = [f"q{idx}" for idx in qubit_indices]

    if isinstance(gate, qiskit_library.Reset):
        return [Reset(q) for q in qubits]

    if isinstance(gate, qiskit_library.XGate):
        return [X(q) for q in qubits]

    if isinstance(gate, qiskit_library.YGate):
        return [Y(q) for q in qubits]

    if isinstance(gate, qiskit_library.ZGate):
        return [Z(q) for q in qubits]

    if isinstance(gate, qiskit_library.HGate):
        return [H(q) for q in qubits]

    if isinstance(gate, qiskit_library.SGate):
        return [S(q) for q in qubits]

    if isinstance(gate, qiskit_library.SdgGate):
        return [SDagger(q) for q in qubits]

    if isinstance(gate, qiskit_library.TGate):
        return [T(q) for q in qubits]

    if isinstance(gate, qiskit_library.TdgGate):
        return [TDagger(q) for q in qubits]

    if isinstance(gate, qiskit_library.SXGate):
        return [Rxy(theta=90.0, phi=0.0, qubit=q) for q in qubits]

    if isinstance(gate, qiskit_library.SXdgGate):
        return [Rxy(theta=-90.0, phi=0.0, qubit=q) for q in qubits]

    if isinstance(gate, qiskit_library.CXGate):
        c, t = qubits[0], qubits[1]
        return [H(t), CZ(qC=c, qT=t), H(t)]

    if isinstance(gate, qiskit_library.CCXGate):
        # Explicitly extract the three qubits for this specific instruction
        c1, c2, t = qubits[0], qubits[1], qubits[2]
        return [
            H(t),
            CZ(qC=c2, qT=t),
            TDagger(t),
            CZ(qC=c1, qT=t),
            T(t),
            CZ(qC=c2, qT=t),
            TDagger(t),
            H(t),
            # Final entangling phase cleanup between the two control lines
            CZ(qC=c1, qT=c2),
            TDagger(c2),
            CZ(qC=c1, qT=c2),
        ]

    if isinstance(gate, qiskit_library.CZGate):
        return [CZ(qC=qubits[0], qT=qubits[1])]

    if isinstance(gate, qiskit_library.SwapGate):
        c, t = qubits[0], qubits[1]
        return [
            H(t),
            CZ(qC=c, qT=t),
            H(t),
            H(c),
            CZ(qC=c, qT=t),
            H(c),
            H(t),
            CZ(qC=c, qT=t),
            H(t),
        ]

    if isinstance(gate, qiskit_library.RXGate):
        theta_deg = float(np.degrees(params[0]))
        return [Rxy(theta_deg, 0, q) for q in qubits]

    if isinstance(gate, qiskit_library.RYGate):
        theta_deg = float(np.degrees(params[0]))
        return [Rxy(theta_deg, 90, q) for q in qubits]

    if isinstance(gate, (qiskit_library.RZGate, qiskit_library.PhaseGate)):
        theta_deg = float(np.degrees(params[0]))
        return [Rz(theta_deg, q) for q in qubits]

    if isinstance(gate, qiskit_library.UGate):
        theta_deg = float(np.degrees(params[0]))
        phi_deg = float(np.degrees(params[1]))
        lam_deg = float(np.degrees(params[2]))

        ops = []
        for q in qubits:
            ops.extend(
                [
                    Rz(theta=lam_deg, qubit=q),
                    Rxy(theta=theta_deg, phi=90.0, qubit=q),
                    Rz(theta=phi_deg, qubit=q),
                ]
            )
        return ops

    if isinstance(gate, qiskit_library.Measure):
        result = []
        extra = acq_kwargs or {}
        for idx in qubit_indices:
            acq_idx = acq_indices.get(idx, 0)
            # Use unique acq_channel per qubit to avoid overlaps
            result.append(
                Measure(
                    f"q{idx}",
                    acq_channel=idx,
                    acq_index=acq_idx,
                    acq_protocol=acq_protocol,
                    **extra,
                )
            )
            # update the measurement count for that qubit to allow for multiple measurements on a qubit
            acq_indices[idx] = acq_idx + 1
        return result

    if isinstance(gate, qiskit.circuit.Delay):
        duration = gate.duration
        unit = gate.unit
        if unit == "s":
            time_ns = duration * 1e9
        elif unit == "ms":
            time_ns = duration * 1e6
        elif unit == "us":
            time_ns = duration * 1e3
        elif unit == "ns":
            time_ns = duration
        else:
            raise ValueError(f"Delay unit '{unit}' is not supported without a backend.")

        duration_s = _to_multiple_of_4(time_ns) * 1e-9
        return [IdlePulse(duration=duration_s)]

    if isinstance(gate, qiskit_library.Barrier):
        return []

    raise ValueError(f"Gate '{name}' is not supported by QbloxExecutor")


def generate_schedule(
    name: str, circuit: QuantumCircuit, shots: int, acq_protocol: str, acq_kwargs: dict
) -> Schedule:
    """Generate a schedule from the given circuit.

    Args:
        name: the name of the schedule
        circuit: the Qiskit circuit specifying shots, circuits, meas_level, etc.
        shots: number of shots for the experiment
        acq_protocol: Acquisition protocol to use when measuring
        acq_kwargs: Additional arguments passed for acquisition.

    Returns:
        the Schedule with all the proper timings
    """
    schedule = Schedule(name=name, repetitions=shots)
    acq_indices = {}

    for instruction in circuit.data:
        parsed_ops = to_qblox_gates(
            circuit=circuit,
            instruction=instruction,
            acq_indices=acq_indices,
            acq_protocol=acq_protocol,
            acq_kwargs=acq_kwargs,
        )

        import qiskit

        is_parallel_op = isinstance(
            instruction.operation,
            (qiskit.circuit.Measure, qiskit.circuit.Delay, qiskit.circuit.Barrier),
        )

        if is_parallel_op and parsed_ops:
            first_op = schedule.add(parsed_ops[0])
            for op in parsed_ops[1:]:
                schedule.add(op, ref_op=first_op, ref_pt="start")
        else:
            for op in parsed_ops:
                schedule.add(op)

    return schedule


def _to_multiple_of_4(time_ns: float) -> int:
    """Convert a time ns to multiple of 4"""
    return int(round(time_ns / 4.0) * 4)
