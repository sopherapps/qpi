"""Unit tests for qpi_client.client module."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest
from qpi_client.client import QPIClient


@pytest.fixture
def client() -> QPIClient:
    return QPIClient("http://localhost:8090", api_token="test-token")


@pytest.fixture
def mock_response() -> MagicMock:
    """Return a mock requests Response."""
    resp = MagicMock()
    resp.raise_for_status = MagicMock()
    resp.json.return_value = {}
    return resp


class TestSubmitJob:
    def test_submit_job_success(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        mock_response.json.return_value = {"job_id": "job-123"}
        with patch.object(
            client._session, "post", return_value=mock_response
        ) as mock_post:
            job_id = client.submit_job([{"circuit": "OPENQASM 3.0;"}])

        assert job_id == "job-123"
        mock_post.assert_called_once()
        args, kwargs = mock_post.call_args
        assert args[0] == "http://localhost:8090/api/jobs"
        assert kwargs["json"]["circuits"] == [{"circuit": "OPENQASM 3.0;"}]
        assert kwargs["json"]["shots"] == 1024
        assert kwargs["json"]["meas_level"] == 2
        assert kwargs["json"]["meas_return"] == "single"

    def test_submit_job_with_options(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        mock_response.json.return_value = {"id": "job-456"}
        with patch.object(client._session, "post", return_value=mock_response):
            job_id = client.submit_job(
                circuits=[{"circuit": "qasm", "shots": 512}],
                shots=2048,
                meas_level=1,
                meas_return="avg",
                qpu_target="qpu-01",
            )

        assert job_id == "job-456"

    def test_submit_job_missing_id_raises(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        mock_response.json.return_value = {"other": "data"}
        with patch.object(client._session, "post", return_value=mock_response):
            with pytest.raises(ValueError, match="did not contain a job ID"):
                client.submit_job([{"circuit": "qasm"}])


class TestGetJob:
    def test_get_job_success(self, client: QPIClient, mock_response: MagicMock) -> None:
        mock_response.json.return_value = {
            "id": "job-123",
            "status": "completed",
            "results": {"counts": {"0x0": 512}},
        }
        with patch.object(
            client._session, "get", return_value=mock_response
        ) as mock_get:
            data = client.get_job("job-123")

        assert data["id"] == "job-123"
        assert data["status"] == "completed"
        mock_get.assert_called_once_with("http://localhost:8090/api/jobs/job-123")


class TestListJobs:
    def test_list_jobs_bare_array(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        mock_response.json.return_value = [
            {"id": "job-1", "status": "pending"},
            {"id": "job-2", "status": "completed"},
        ]
        with patch.object(client._session, "get", return_value=mock_response):
            jobs = client.list_jobs()

        assert len(jobs) == 2
        assert jobs[0]["id"] == "job-1"

    def test_list_jobs_wrapped(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        mock_response.json.return_value = {
            "jobs": [{"id": "job-3", "status": "running"}]
        }
        with patch.object(client._session, "get", return_value=mock_response):
            jobs = client.list_jobs()

        assert len(jobs) == 1
        assert jobs[0]["id"] == "job-3"


class TestCancelJob:
    def test_cancel_job_success(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        mock_response.json.return_value = {"status": "cancelled"}
        with patch.object(
            client._session, "post", return_value=mock_response
        ) as mock_post:
            data = client.cancel_job("job-123")

        assert data["status"] == "cancelled"
        mock_post.assert_called_once_with(
            "http://localhost:8090/api/jobs/job-123/cancel"
        )


class TestAuthHeader:
    def test_api_token_header_set(self) -> None:
        client = QPIClient("http://localhost:8090", api_token="secret")
        assert client._session.headers["X-API-Token"] == "secret"

    def test_no_token_header_when_none(self) -> None:
        client = QPIClient("http://localhost:8090")
        assert "X-API-Token" not in client._session.headers


class TestContextManager:
    def test_context_manager_closes_session(self) -> None:
        client = QPIClient("http://localhost:8090")
        with patch.object(client._session, "close") as mock_close:
            with client:
                pass
            mock_close.assert_called_once()
