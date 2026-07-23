"""Shared utilities for building Qiskit-compatible result dictionaries."""


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
        dict: Qiskit-compatible result dict with 'circuit_results' always present.
    """
    first = experiment_results[0] if experiment_results else {}
    out = {
        "shots": first.get("shots", 0),
        "backend": backend,
        "success": True,
        "circuit_results": experiment_results,
    }

    if "counts" in first:
        out["counts"] = first["counts"]
    if "memory" in first:
        out["memory"] = first["memory"]

    return out


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
