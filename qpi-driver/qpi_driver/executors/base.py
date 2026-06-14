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
    acq_rotation: float | None = None
    acq_threshold: float | None = None

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
        acq_rotation = data.get("acq_rotation")
        if acq_rotation is not None:
            acq_rotation = float(acq_rotation)
        acq_threshold = data.get("acq_threshold")
        if acq_threshold is not None:
            acq_threshold = float(acq_threshold)

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
            acq_rotation=acq_rotation,
            acq_threshold=acq_threshold,
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

    @abstractmethod
    def process_result(self, dataset: xr.Dataset, job_id: str) -> dict:
        """Convert a raw xr.Dataset from execute() into a Qiskit-compatible result dict.

        Each executor knows its own data format and how to:
        - Perform state discrimination (if meas_level=2)
        - Return IQ memory (if meas_level=1)
        - Return raw traces (if meas_level=0)
        - Handle meas_return averaging

        Args:
            dataset: The xr.Dataset returned by execute().
            job_id: The unique ID of the quantum job.

        Returns:
            dict: Qiskit-compatible result dict with keys like 'counts', 'memory',
                  'shots', 'backend', 'success', etc.
        """
        pass

    def close(self) -> None:
        """Release resources."""
        pass
