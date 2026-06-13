"""Low-level HTTP client for the QPI orchestrator REST API.

This module provides :class:`QPIClient`, a thin wrapper around ``requests.Session``
that handles authentication and serialisation for every QPI API endpoint.
"""

from __future__ import annotations

from typing import Any

import requests


class QPIClient:
    """Low-level HTTP wrapper for the QPI orchestrator API.

    Args:
        base_url: Root URL of the QPI orchestrator (e.g. ``"http://localhost:8090"``).
        api_token: Optional API token used for authentication via the
            ``X-API-Token`` header. When *None*, no token header is sent
            (useful for cookie/JWT-based auth in browser contexts).
    """

    def __init__(self, base_url: str, api_token: str | None = None) -> None:
        self.base_url: str = base_url.rstrip("/")
        self.api_token: str | None = api_token
        self._session: requests.Session = requests.Session()
        self._session.headers["Content-Type"] = "application/json"
        if api_token:
            self._session.headers["X-API-Token"] = api_token

    # -- public API ----------------------------------------------------------

    def submit_job(
        self,
        circuits: list[dict[str, Any]],
        shots: int = 1024,
        meas_level: int = 2,
        meas_return: str = "single",
        qpu_target: str = "",
    ) -> str:
        """Submit a quantum job to the orchestrator.

        Args:
            circuits: A list of circuit payload dicts.  Each dict **must**
                contain a ``"circuit"`` key whose value is an OpenQASM 3
                string.  Optional keys: ``"parameter_values"``, ``"shots"``.
            shots: Default number of shots for every circuit.
            meas_level: Measurement level (``2`` = classified bits).
            meas_return: ``"single"`` or ``"avg"``.
            qpu_target: Optional QPU routing hint.

        Returns:
            The server-assigned job ID as a string.

        Raises:
            requests.HTTPError: If the server returns a non-2xx status.
        """
        payload: dict[str, Any] = {
            "circuits": circuits,
            "shots": shots,
            "meas_level": meas_level,
            "meas_return": meas_return,
        }
        if qpu_target:
            payload["qpu_target"] = qpu_target

        resp = self._session.post(f"{self.base_url}/api/jobs", json=payload)
        resp.raise_for_status()
        data = resp.json()

        # The orchestrator may return the ID at the top level or nested.
        job_id: str = data.get("id") or data.get("job_id", "")
        if not job_id:
            raise ValueError(f"Server response did not contain a job ID: {data!r}")
        return job_id

    def get_job(self, job_id: str) -> dict[str, Any]:
        """Retrieve full details for *job_id*.

        Returns:
            A dict with at least ``"id"``, ``"status"``, ``"payload"``,
            ``"results"``, ``"created"``, and ``"updated"`` keys.

        Raises:
            requests.HTTPError: If the server returns a non-2xx status.
        """
        resp = self._session.get(f"{self.base_url}/api/jobs/{job_id}")
        resp.raise_for_status()
        return resp.json()

    def list_jobs(self) -> list[dict[str, Any]]:
        """List all jobs belonging to the authenticated user.

        Returns:
            A list of job-record dicts.

        Raises:
            requests.HTTPError: If the server returns a non-2xx status.
        """
        resp = self._session.get(f"{self.base_url}/api/jobs")
        resp.raise_for_status()
        data = resp.json()
        # The response might be a bare list or wrapped in {"jobs": [...]}.
        if isinstance(data, list):
            return data
        return data.get("jobs", [])

    def cancel_job(self, job_id: str) -> dict[str, Any]:
        """Request cancellation of *job_id*.

        Returns:
            The updated job-record dict.

        Raises:
            requests.HTTPError: If the server returns a non-2xx status.
        """
        resp = self._session.post(f"{self.base_url}/api/jobs/{job_id}/cancel")
        resp.raise_for_status()
        return resp.json()

    # -- high-level helpers --------------------------------------------------

    def get_backend(self, name: str = "qpi") -> "QPIBackend":
        """Return a :class:`QPIBackend` handle for the named QPU.

        Args:
            name: Backend / QPU name (e.g. ``"mock"``, ``"qiskit_aer"``).

        Returns:
            A configured :class:`QPIBackend` instance bound to this client.
        """
        from qpi_client.provider import QPIBackend

        return QPIBackend(self, name=name)

    def job(self, job_id: str) -> "QPIJob":
        """Retrieve an existing job by ID.

        Args:
            job_id: The server-assigned job ID.

        Returns:
            A :class:`QPIJob` handle (backend will be *None*).
        """
        from qpi_client.provider import QPIJob

        return QPIJob(backend=None, job_id=job_id, client=self)

    # -- QPU discovery -------------------------------------------------------

    def list_qpus(self) -> list[dict[str, Any]]:
        """List all online QPUs.

        Returns:
            A list of QPU record dicts.
        """
        resp = self._session.get(f"{self.base_url}/api/qpus")
        resp.raise_for_status()
        return resp.json()

    def get_qpu(self, name: str) -> dict[str, Any]:
        """Retrieve a single QPU by name.

        Args:
            name: The QPU's unique name.

        Returns:
            A QPU record dict.
        """
        resp = self._session.get(f"{self.base_url}/api/qpus/{name}")
        resp.raise_for_status()
        return resp.json()

    # -- lifecycle -----------------------------------------------------------

    def close(self) -> None:
        """Close the underlying HTTP session."""
        self._session.close()

    def __enter__(self) -> "QPIClient":
        return self

    def __exit__(self, *exc: object) -> None:
        self.close()
