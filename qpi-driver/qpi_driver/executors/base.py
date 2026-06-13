import uuid
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any

import xarray as xr


@dataclass
class CircuitPayload:
    """A single circuit within a batch job."""

    circuit: str  # QASM string
    parameter_values: list[list[float]] | None = None  # Optional parameter bindings
    shots: int | None = None  # Per-circuit override


@dataclass
class JobPayload:
    circuits: list[CircuitPayload]
    id: str = field(default_factory=lambda: str(uuid.uuid4()))
    shots: int = 1024
    meas_level: int = 2  # 2=counts, 1=kerneled IQ, 0=raw IQ
    meas_return: str = "single"  # "single" or "avg"

    @property
    def qasm(self) -> str:
        """Backward-compatible accessor: returns the first circuit's QASM string."""
        return self.circuits[0].circuit

    def __post_init__(self):
        if not self.id.strip():
            raise ValueError("id cannot be empty or just whitespace.")

        if not self.circuits:
            raise ValueError("circuits list cannot be empty.")

        for i, cp in enumerate(self.circuits):
            if not cp.circuit.strip():
                raise ValueError(f"Circuit at index {i} has an empty circuit string.")

        if self.meas_level not in (0, 1, 2):
            raise ValueError(f"meas_level must be 0, 1, or 2, got {self.meas_level}")

        if self.meas_return not in ("single", "avg"):
            raise ValueError(
                f"meas_return must be 'single' or 'avg', got '{self.meas_return}'"
            )

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "JobPayload":
        shots = data.get("shots") or 1024
        identifier = data.get("id") or str(uuid.uuid4())
        meas_level = data.get("meas_level", 2)
        meas_return = data.get("meas_return", "single")

        # New-style: list of circuit dicts under "circuits"
        raw_circuits = data.get("circuits")
        if raw_circuits and isinstance(raw_circuits, list):
            circuits = []
            for entry in raw_circuits:
                if isinstance(entry, dict):
                    circ_str = (
                        entry.get("circuit")
                        or entry.get("qasm")
                        or entry.get("circuit_qasm")
                        or ""
                    )
                    param_vals = entry.get("parameter_values")
                    per_circuit_shots = entry.get("shots")
                    circuits.append(
                        CircuitPayload(
                            circuit=circ_str,
                            parameter_values=param_vals,
                            shots=per_circuit_shots,
                        )
                    )
                elif isinstance(entry, str):
                    # Plain QASM string in list
                    circuits.append(CircuitPayload(circuit=entry))
                else:
                    raise ValueError(
                        f"Invalid circuit entry type: {type(entry)}. "
                        "Expected dict or string."
                    )
            if not circuits:
                raise ValueError("No circuits provided in payload")
        else:
            # Old-style: single QASM string
            qasm = (
                data.get("qasm")
                or data.get("circuit_qasm")
                or data.get("circuit")
                or ""
            )
            if not qasm:
                raise ValueError("No QASM string/circuit provided in payload")
            circuits = [CircuitPayload(circuit=qasm)]

        return cls(
            circuits=circuits,
            shots=shots,
            id=identifier,
            meas_level=meas_level,
            meas_return=meas_return,
        )


class Executor(ABC):
    def __init__(self, name: str = "executor", **kwargs: Any) -> None:
        self.name = name

    @abstractmethod
    def execute(self, payload: JobPayload) -> xr.Dataset:
        """Execute the quantum circuit/instructions payload.

        Args:
            payload: JobPayload object containing circuit QASM, qubit count, shots, etc.

        Returns:
            xr.Dataset: Dataset mimicking the raw measurement counts and frequencies.
        """
        pass

    def close(self) -> None:
        """Release resources."""
        pass
