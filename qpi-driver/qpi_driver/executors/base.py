from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Any

import xarray as xr


@dataclass
class JobPayload:
    qasm: str
    shots: int = 1024
    n_qubits: int = 2

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "JobPayload":
        qasm = data.get("qasm") or data.get("circuit_qasm") or data.get("circuit") or ""
        if not qasm:
            raise ValueError("No QASM string/circuit provided in payload")
        shots = data.get("shots") or 1024
        n_qubits = data.get("n_qubits") or 2
        return cls(qasm=qasm, shots=shots, n_qubits=n_qubits)



class Executor(ABC):
    def __init__(self, **kwargs: Any) -> None:
        pass

    @abstractmethod
    def execute(self, payload: JobPayload) -> xr.Dataset:
        """Execute the quantum circuit/instructions payload.

        Args:
            payload: JobPayload object containing circuit QASM, qubit count, shots, etc.

        Returns:
            xr.Dataset: Dataset mimicking the raw measurement counts and frequencies.
        """
        pass

