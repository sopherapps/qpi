"""Qiskit provider integration for the QPI platform.

This module exposes :class:`QPIBackend` (a Qiskit ``BackendV2``) and
:class:`QPIJob` (a Qiskit ``JobV1``) so that QPI can be used as a
drop-in Qiskit execution target::

    from qiskit.circuit import QuantumCircuit
    from qpi_client import QPIClient, QPIBackend

    client = QPIClient("http://localhost:8090", api_token="tok")
    backend = QPIBackend(client, num_qubits=5)

    qc = QuantumCircuit(2, 2)
    qc.h(0)
    qc.cx(0, 1)
    qc.measure([0, 1], [0, 1])

    job = backend.run(qc, shots=4096)
    result = job.result(timeout=120)
    print(result.get_counts())
"""

from __future__ import annotations

import time
from typing import Any, Sequence

from qiskit.circuit import QuantumCircuit
from qiskit.providers import BackendV2, JobStatus, JobV1, Options
from qiskit.qasm3 import dumps as qasm3_dumps
from qiskit.result import Result
from qiskit.result.models import ExperimentResult, ExperimentResultData
from qiskit.transpiler import Target

from qpi_client.client import QPIClient

# ---------------------------------------------------------------------------
# QPIJob
# ---------------------------------------------------------------------------


class QPIJob(JobV1):
    """A Qiskit-compatible job handle backed by the QPI REST API.

    Instances are created by :meth:`QPIBackend.run`; you should not need to
    instantiate this class directly.
    """

    def __init__(
        self,
        backend: "QPIBackend",
        job_id: str,
        client: QPIClient,
        **kwargs: Any,
    ) -> None:
        super().__init__(backend, job_id, **kwargs)
        self._client = client
        self._result: Result | None = None

    # -- JobV1 interface -----------------------------------------------------

    def submit(self) -> None:
        """No-op — the job was already submitted by the backend."""

    def result(
        self,
        timeout: float | None = None,
        wait: float = 5.0,
    ) -> Result:
        """Block until the job completes and return a :class:`qiskit.result.Result`.

        Args:
            timeout: Maximum seconds to wait. *None* means wait indefinitely.
            wait: Polling interval in seconds.

        Returns:
            A Qiskit :class:`Result` object populated with counts and
            (optionally) memory from the QPI server response.

        Raises:
            TimeoutError: If the job does not finish within *timeout* seconds.
            RuntimeError: If the job fails or is cancelled on the server.
        """
        if self._result is not None:
            return self._result

        start = time.monotonic()
        while True:
            data = self._client.get_job(self.job_id())
            status = data.get("status", "")
            if status in ("completed", "failed", "cancelled"):
                break
            if timeout is not None and (time.monotonic() - start) > timeout:
                raise TimeoutError(
                    f"Job {self.job_id()} did not complete within {timeout}s"
                )
            time.sleep(wait)

        if status == "failed":
            error_msg = ""
            results_data = data.get("results")
            if isinstance(results_data, dict):
                error_msg = results_data.get("error", "")
            raise RuntimeError(f"Job {self.job_id()} failed: {error_msg}")

        if status == "cancelled":
            raise RuntimeError(f"Job {self.job_id()} was cancelled")

        self._result = self._build_result(data)
        return self._result

    def status(self) -> JobStatus:
        """Return the current server-side status of the job."""
        data = self._client.get_job(self.job_id())
        _STATUS_MAP: dict[str, JobStatus] = {
            "pending": JobStatus.QUEUED,
            "queued": JobStatus.QUEUED,
            "running": JobStatus.RUNNING,
            "completed": JobStatus.DONE,
            "failed": JobStatus.ERROR,
            "cancelled": JobStatus.CANCELLED,
        }
        return _STATUS_MAP.get(data.get("status", ""), JobStatus.ERROR)

    def cancel(self) -> None:
        """Request cancellation of this job on the server."""
        self._client.cancel_job(self.job_id())

    # -- internal helpers ----------------------------------------------------

    def _build_result(self, data: dict[str, Any]) -> Result:
        """Construct a :class:`qiskit.result.Result` from the API response.

        The server ``results`` payload may be:
        * A dict with a top-level ``"circuit_results"`` list (one entry per
          submitted circuit).
        * A single dict with ``"counts"``/``"hex_counts"``/``"memory"`` keys
          when only one circuit was submitted.
        * ``None`` (edge-case) — we still return a valid *Result* with no
          experiment data.
        """
        results_payload: Any = data.get("results") or {}

        # Normalise to a list of per-circuit result dicts.
        if isinstance(results_payload, dict):
            circuit_results: list[dict[str, Any]] = results_payload.get(
                "circuit_results", []
            )
            if not circuit_results:
                # Treat the whole dict as a single-circuit result.
                circuit_results = [results_payload]
        elif isinstance(results_payload, list):
            circuit_results = results_payload
        else:
            circuit_results = []

        experiment_results: list[ExperimentResult] = []
        for idx, cr in enumerate(circuit_results):
            counts = cr.get("counts") or cr.get("hex_counts") or {}
            # Ensure keys are hex-string formatted ("0x…").
            hex_counts: dict[str, int] = {}
            for key, val in counts.items():
                if isinstance(key, int):
                    hex_counts[hex(key)] = int(val)
                elif key.startswith("0x") or key.startswith("0X"):
                    hex_counts[key] = int(val)
                else:
                    # Assume binary string — convert to hex.
                    try:
                        hex_counts[hex(int(key, 2))] = int(val)
                    except ValueError:
                        hex_counts[key] = int(val)

            exp_data = ExperimentResultData(
                counts=hex_counts,
                memory=cr.get("memory"),
            )

            experiment_results.append(
                ExperimentResult(
                    shots=cr.get(
                        "shots", sum(hex_counts.values()) if hex_counts else 0
                    ),
                    success=True,
                    data=exp_data,
                    header=cr.get("header"),
                )
            )

        return Result(
            backend_name=self.backend().name if self.backend() else "qpi",
            backend_version="0.1.0",
            qobj_id=None,
            job_id=self.job_id(),
            success=True,
            results=experiment_results,
        )


# ---------------------------------------------------------------------------
# QPIBackend
# ---------------------------------------------------------------------------


class QPIBackend(BackendV2):
    """A Qiskit ``BackendV2`` that submits circuits to the QPI orchestrator.

    Args:
        client: An authenticated :class:`QPIClient` instance.
        name: Human-readable backend name (default ``"qpi"``).
        num_qubits: Number of qubits the backend advertises.
        **kwargs: Forwarded to :class:`BackendV2.__init__`.
    """

    def __init__(
        self,
        client: QPIClient,
        name: str = "qpi",
        num_qubits: int = 5,
        **kwargs: Any,
    ) -> None:
        super().__init__(name=name, **kwargs)
        self._client = client
        self._num_qubits = num_qubits
        self._target = Target(num_qubits=num_qubits)
        self._options = Options()
        self._options.update_options(
            shots=1024,
            meas_level=2,
            meas_return="single",
        )

    # -- BackendV2 required properties ---------------------------------------

    @property
    def target(self) -> Target:
        """Return the transpiler :class:`Target` for this backend."""
        return self._target

    @property
    def max_circuits(self) -> int | None:
        """No server-side limit on the number of circuits per job."""
        return None

    @classmethod
    def _default_options(cls) -> Options:
        return Options(shots=1024, meas_level=2, meas_return="single")

    # -- execution -----------------------------------------------------------

    def run(
        self,
        circuits: QuantumCircuit | Sequence[QuantumCircuit],
        **kwargs: Any,
    ) -> QPIJob:
        """Transpile, serialise, and submit *circuits* to QPI.

        Args:
            circuits: A single :class:`QuantumCircuit` or a list thereof.
            **kwargs: Override options such as ``shots``, ``meas_level``,
                ``meas_return``, ``parameter_values``.

        Returns:
            A :class:`QPIJob` handle that can be polled or awaited.
        """
        if isinstance(circuits, QuantumCircuit):
            circuits = [circuits]

        shots: int = kwargs.get("shots", self._options.get("shots", 1024))
        meas_level: int = kwargs.get("meas_level", self._options.get("meas_level", 2))
        meas_return: str = kwargs.get(
            "meas_return", self._options.get("meas_return", "single")
        )
        parameter_values: list[dict[Any, float]] | None = kwargs.get("parameter_values")

        circuit_payloads: list[dict[str, Any]] = []
        for idx, qc in enumerate(circuits):
            # Handle parameter binding
            if parameter_values and idx < len(parameter_values):
                pv = parameter_values[idx]
                if pv:
                    bound_qc = qc.assign_parameters(pv)
                    qasm_str = qasm3_dumps(bound_qc)
                    # Order values according to qc.parameters for the payload
                    ordered_values = [float(pv[p]) for p in qc.parameters]
                    circuit_payloads.append(
                        {
                            "circuit": qasm_str,
                            "parameter_values": [ordered_values],
                        }
                    )
                    continue

            qasm_str = qasm3_dumps(qc)
            circuit_payloads.append({"circuit": qasm_str})

        job_id = self._client.submit_job(
            circuits=circuit_payloads,
            shots=shots,
            meas_level=meas_level,
            meas_return=meas_return,
        )
        return QPIJob(self, job_id, self._client)
