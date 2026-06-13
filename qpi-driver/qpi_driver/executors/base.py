import uuid
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any

import xarray as xr


@dataclass
class JobPayload:
    qasm: str
    id: str = field(default_factory=lambda: str(uuid.uuid4()))
    shots: int = 1024
    n_qubits: int = 2

    def __post_init__(self):
        if not self.id.strip():
            raise ValueError("id cannot be empty or just whitespace.")

        if not self.qasm.strip():
            raise ValueError("qasm cannot be empty or just whitespace.")

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "JobPayload":
        qasm = data.get("qasm") or data.get("circuit_qasm") or data.get("circuit") or ""
        if not qasm:
            raise ValueError("No QASM string/circuit provided in payload")
        shots = data.get("shots") or 1024
        n_qubits = data.get("n_qubits") or 2
        identifier = data.get("id") or str(uuid.uuid4())
        return cls(qasm=qasm, shots=shots, n_qubits=n_qubits, id=identifier)


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
