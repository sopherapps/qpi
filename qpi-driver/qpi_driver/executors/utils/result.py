"""Shared utilities for building Qiskit-compatible result dictionaries."""

from qiskit.result import Result
from qiskit.result.models import ExperimentResult, ExperimentResultData


def build_experiment_result(
    result_data: dict, shots: int, name: str = "qpi_job"
) -> ExperimentResult:
    """Build a single Qiskit ExperimentResult from a result data dict.

    Args:
        result_data: Dict with either 'counts' (binary str -> int) or 'memory' (IQ list).
        shots: Number of shots.
        name: Experiment name.

    Returns:
        ExperimentResult object.
    """
    if "memory" in result_data:
        expt_data = ExperimentResultData(memory=result_data["memory"])
    else:
        expt_data = ExperimentResultData(
            counts={hex(int(s, 2)): c for s, c in result_data["counts"].items()}
        )
    return ExperimentResult(
        shots=shots,
        success=True,
        data=expt_data,
        status="DONE",
        name=name,
    )


def build_qiskit_result(
    experiment_results: list[dict],
    job_id: str,
    backend: str,
) -> dict:
    """Build a Qiskit-compatible result dict from a list of per-circuit experiment results.

    Each experiment result dict should have:
    - 'counts': dict[str, int] (binary string keys) OR 'memory': list (IQ data)
    - 'shots': int

    Args:
        experiment_results: List of per-circuit result dicts.
        job_id: The unique job ID.
        backend: Backend name string.

    Returns:
        dict: Qiskit-compatible result dict with top-level keys and optional 'circuit_results'.
    """
    exp_result_objs = [
        build_experiment_result(er, er["shots"]) for er in experiment_results
    ]

    result_obj = Result(
        backend_name=backend,
        backend_version="1.0.0",
        qobj_id=job_id,
        job_id=job_id,
        success=True,
        results=exp_result_objs,
    )

    first = experiment_results[0]
    out = {
        "shots": first["shots"],
        "backend": backend,
        "success": True,
    }

    if len(experiment_results) > 1:
        out["circuit_results"] = experiment_results

    if "counts" in first:
        out["counts"] = first["counts"]
        out["hex_counts"] = result_obj.get_counts(0)
    if "memory" in first:
        out["memory"] = first["memory"]

    return out


def counts_to_hex(counts_dict: dict[str, int]) -> dict[str, int]:
    """Convert binary-string-keyed counts to hex-string-keyed counts.

    Args:
        counts_dict: Dict mapping binary strings (e.g. '01') to counts.

    Returns:
        Dict mapping hex strings (e.g. '0x1') to counts.
    """
    return {hex(int(s, 2)): c for s, c in counts_dict.items()}


def iq_memory_avg(
    memory: list[list[list[float]]], n_qubits: int
) -> list[list[list[float]]]:
    """Average IQ memory data across shots.

    Args:
        memory: Per-shot IQ data, shape (n_shots, n_qubits, 2).
        n_qubits: Number of qubits.

    Returns:
        Averaged IQ data, shape (1, n_qubits, 2).
    """
    import numpy as np

    arr = np.array(memory)  # (shots, n_qubits, 2)
    avg = arr.mean(axis=0).tolist()  # (n_qubits, 2)
    return [avg]
