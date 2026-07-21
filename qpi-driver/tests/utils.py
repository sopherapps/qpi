import json
from pathlib import Path
from typing import Any, Iterable, Sequence

import numpy as np
import yaml

_FIXTURES_PATH = Path(__file__).parent / "fixtures"


def load_json_fixture(fixture_relative_path: str) -> Any:
    """Load a JSON fixture from the tests/fixtures directory."""
    full_path = _FIXTURES_PATH / fixture_relative_path
    with open(full_path, "r") as f:
        return json.load(f)


def load_yaml_fixture(fixture_relative_path: str) -> Any:
    """Load a YAML fixture from the tests/fixtures directory."""
    full_path = _FIXTURES_PATH / fixture_relative_path
    with open(full_path, "r") as f:
        return yaml.safe_load(f)


def operations_to_unitary(
    operations: Iterable[Any], qubits: Sequence[str]
) -> np.ndarray:
    """Build the unitary of a list of quantify/qblox gate operations.

    Each operation's own ``gate_info`` unitary is embedded onto the global qubit
    space, so the result reflects exactly what the emitted gates mean. Qubits are
    ordered as given, with the first qubit the most significant bit.
    """
    order = list(qubits)
    position = {qubit: bit for bit, qubit in enumerate(order)}
    num_qubits = len(order)
    unitary = np.eye(2**num_qubits, dtype=complex)

    for operation in operations:
        gate_info = operation.data["gate_info"]
        gate_unitary = np.asarray(gate_info["unitary"], dtype=complex)
        targets = [position[qubit] for qubit in _gate_qubits(gate_info)]
        unitary = _embed(gate_unitary, targets, num_qubits) @ unitary

    return unitary


def equal_up_to_global_phase(a: np.ndarray, b: np.ndarray, atol: float = 1e-9) -> bool:
    """Whether two matrices are equal up to a global phase factor."""
    pivot = np.unravel_index(np.argmax(np.abs(a)), a.shape)
    if abs(a[pivot]) < atol:
        return False
    phase = b[pivot] / a[pivot]
    return bool(np.allclose(a * phase, b, atol=atol))


def toffoli_unitary() -> np.ndarray:
    """Unitary of a Toffoli (CCX) with qubit order (control, control, target)."""
    unitary = np.eye(8, dtype=complex)
    unitary[[6, 7]] = unitary[[7, 6]]
    return unitary


def _gate_qubits(gate_info: dict) -> list[str]:
    return gate_info.get("device_elements") or gate_info["qubits"]


def _embed(
    gate_unitary: np.ndarray, targets: Sequence[int], num_qubits: int
) -> np.ndarray:
    """Embed a gate acting on ``targets`` into the full ``num_qubits`` space.

    ``targets`` are ordered to match the tensor factors of ``gate_unitary``, with
    the first target the most significant qubit of the gate.
    """
    dimension = 2**num_qubits
    num_targets = len(targets)
    embedded = np.zeros((dimension, dimension), dtype=complex)

    for column in range(dimension):
        column_bits = [
            (column >> (num_qubits - 1 - qubit)) & 1 for qubit in range(num_qubits)
        ]
        gate_column = 0
        for qubit in targets:
            gate_column = (gate_column << 1) | column_bits[qubit]

        for gate_row in range(2**num_targets):
            amplitude = gate_unitary[gate_row, gate_column]
            if amplitude == 0:
                continue
            row_bits = list(column_bits)
            for offset, qubit in enumerate(targets):
                row_bits[qubit] = (gate_row >> (num_targets - 1 - offset)) & 1
            row = 0
            for bit in row_bits:
                row = (row << 1) | bit
            embedded[row, column] += amplitude

    return embedded
