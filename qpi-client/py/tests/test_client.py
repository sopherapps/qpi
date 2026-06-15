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
            "duration": 1.23,
        }
        with patch.object(
            client._session, "get", return_value=mock_response
        ) as mock_get:
            data = client.get_job("job-123")

        assert data["id"] == "job-123"
        assert data["status"] == "completed"
        assert data["duration"] == 1.23
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


class TestGetBackend:
    def test_get_backend_returns_backend(self, client: QPIClient) -> None:
        from qpi_client.provider import QPIBackend

        with patch.object(
            client, "get_qpu", return_value={"name": "mock", "num_qubits": 5}
        ):
            backend = client.get_backend(name="mock")

        assert isinstance(backend, QPIBackend)
        assert backend.name == "mock"

    def test_get_backend_default_name(self, client: QPIClient) -> None:
        with patch.object(
            client, "get_qpu", return_value={"name": "qpi", "num_qubits": 5}
        ):
            backend = client.get_backend()

        assert backend.name == "qpi"


class TestJob:
    def test_job_returns_job(self, client: QPIClient) -> None:
        job = client.job("job-789")
        from qpi_client.provider import QPIJob

        assert isinstance(job, QPIJob)
        assert job.job_id() == "job-789"


class TestQPU:
    def test_list_qpus(self, client: QPIClient, mock_response: MagicMock) -> None:
        mock_response.json.return_value = [
            {"id": "qpu-1", "name": "mock", "num_qubits": 5},
        ]
        with patch.object(
            client._session, "get", return_value=mock_response
        ) as mock_get:
            qpus = client.list_qpus()

        assert len(qpus) == 1
        assert qpus[0]["name"] == "mock"
        mock_get.assert_called_once_with("http://localhost:8090/api/qpus")

    def test_get_qpu(self, client: QPIClient, mock_response: MagicMock) -> None:
        mock_response.json.return_value = {
            "id": "mock",
            "name": "mock",
            "num_qubits": 5,
        }
        with patch.object(
            client._session, "get", return_value=mock_response
        ) as mock_get:
            qpu = client.get_qpu("mock")

        assert qpu["name"] == "mock"
        assert qpu["num_qubits"] == 5
        mock_get.assert_called_once_with("http://localhost:8090/api/qpus/mock")


class TestNewClientMethods:
    def test_create_qpu(self, client: QPIClient, mock_response: MagicMock) -> None:
        mock_response.json.return_value = {"id": "qpu-123", "access_token": "qpi_abc"}
        with patch.object(
            client._session, "post", return_value=mock_response
        ) as mock_post:
            resp = client.create_qpu("qpu-02", executor_type="mock")

        assert resp == {"id": "qpu-123", "access_token": "qpi_abc"}
        mock_post.assert_called_once()
        args, kwargs = mock_post.call_args
        assert args[0] == "http://localhost:8090/api/op/qpus/create"
        assert kwargs["json"] == {
            "name": "qpu-02",
            "executor_type": "mock",
        }

    def test_connect_qpu(self, client: QPIClient, mock_response: MagicMock) -> None:
        mock_response.json.return_value = {
            "status": "success",
            "nng_command_port": 6000,
        }
        with patch.object(
            client._session, "post", return_value=mock_response
        ) as mock_post:
            resp = client.connect_qpu("qpu-02", "token123", executor_type="mock")

        assert resp["status"] == "success"
        mock_post.assert_called_once()
        args, kwargs = mock_post.call_args
        assert args[0] == "http://localhost:8090/api/op/qpus/connect"
        assert kwargs["json"] == {
            "name": "qpu-02",
            "access_token": "token123",
            "executor_type": "mock",
        }

    def test_toggle_qpu(self, client: QPIClient, mock_response: MagicMock) -> None:
        mock_response.json.return_value = {"success": True}
        with patch.object(
            client._session, "post", return_value=mock_response
        ) as mock_post:
            resp = client.toggle_qpu("qpu-123", True)

        assert resp == {"success": True}
        mock_post.assert_called_once_with(
            "http://localhost:8090/api/op/qpu/toggle",
            json={"id": "qpu-123", "enabled": True},
        )

    def test_list_notifications(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        mock_response.json.return_value = {"items": [{"id": "n1"}]}
        with patch.object(
            client._session, "get", return_value=mock_response
        ) as mock_get:
            resp = client.list_notifications()

        assert resp == [{"id": "n1"}]
        mock_get.assert_called_once_with(
            "http://localhost:8090/api/collections/notifications/records"
        )

    def test_dismiss_notification_direct(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        mock_response.json.return_value = {"status": "dismissed"}
        with patch.object(
            client._session, "post", return_value=mock_response
        ) as mock_post:
            resp = client.dismiss_notification("n1")

        assert resp == {"status": "dismissed"}
        mock_post.assert_called_once_with(
            "http://localhost:8090/api/notifications/n1/dismiss"
        )

    def test_time_slots(self, client: QPIClient, mock_response: MagicMock) -> None:
        # list
        mock_response.json.return_value = {"items": [{"id": "s1"}]}
        with patch.object(
            client._session, "get", return_value=mock_response
        ) as mock_get:
            assert client.list_time_slots() == [{"id": "s1"}]
            mock_get.assert_called_once_with(
                "http://localhost:8090/api/collections/time_slots/records"
            )

        # create
        mock_response.json.return_value = {"id": "s1"}
        with patch.object(
            client._session, "post", return_value=mock_response
        ) as mock_post:
            assert client.create_time_slot("start", "end") == {"id": "s1"}
            mock_post.assert_called_once_with(
                "http://localhost:8090/api/collections/time_slots/records",
                json={"start_time": "start", "end_time": "end"},
            )

        # update
        mock_response.json.return_value = {"id": "s1", "start_time": "start2"}
        with patch.object(
            client._session, "patch", return_value=mock_response
        ) as mock_patch:
            assert client.update_time_slot("s1", start_time="start2") == {
                "id": "s1",
                "start_time": "start2",
            }
            mock_patch.assert_called_once_with(
                "http://localhost:8090/api/collections/time_slots/records/s1",
                json={"start_time": "start2"},
            )

        # delete
        mock_response.json.return_value = None
        with patch.object(
            client._session, "delete", return_value=mock_response
        ) as mock_delete:
            client.delete_time_slot("s1")
            mock_delete.assert_called_once_with(
                "http://localhost:8090/api/collections/time_slots/records/s1"
            )

    def test_time_requests(self, client: QPIClient, mock_response: MagicMock) -> None:
        # list
        mock_response.json.return_value = {"items": [{"id": "tr1"}]}
        with patch.object(
            client._session, "get", return_value=mock_response
        ) as mock_get:
            assert client.list_time_requests() == [{"id": "tr1"}]
            mock_get.assert_called_once_with(
                "http://localhost:8090/api/collections/qpu_time_requests/records"
            )

        # create
        mock_response.json.return_value = {"id": "tr1"}
        with patch.object(
            client._session, "post", return_value=mock_response
        ) as mock_post:
            assert client.create_time_request(100, "reason") == {"id": "tr1"}
            mock_post.assert_called_once_with(
                "http://localhost:8090/api/collections/qpu_time_requests/records",
                json={"seconds": 100, "requested_reason": "reason"},
            )

        # update
        mock_response.json.return_value = {"id": "tr1", "status": "approved"}
        with patch.object(
            client._session, "patch", return_value=mock_response
        ) as mock_patch:
            assert client.update_time_request("tr1", "approved") == {
                "id": "tr1",
                "status": "approved",
            }
            mock_patch.assert_called_once_with(
                "http://localhost:8090/api/collections/qpu_time_requests/records/tr1",
                json={"status": "approved"},
            )

    def test_admin_user_quota(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        # list users
        mock_response.json.return_value = {"items": [{"id": "u1"}]}
        with patch.object(
            client._session, "get", return_value=mock_response
        ) as mock_get:
            assert client.list_users() == [{"id": "u1"}]
            mock_get.assert_called_once_with(
                "http://localhost:8090/api/collections/users/records"
            )

        # allocate
        mock_response.json.return_value = {"id": "u1", "qpu_seconds": 500}
        with patch.object(
            client._session, "patch", return_value=mock_response
        ) as mock_patch:
            assert client.allocate_qpu_time("u1", 500) == {
                "id": "u1",
                "qpu_seconds": 500,
            }
            mock_patch.assert_called_once_with(
                "http://localhost:8090/api/admin/users/u1",
                json={"qpu_seconds": 500},
            )

    def test_auth_with_password(
        self, client: QPIClient, mock_response: MagicMock
    ) -> None:
        mock_response.json.return_value = {"token": "jwt123"}
        with patch.object(
            client._session, "post", return_value=mock_response
        ) as mock_post:
            resp = client.auth_with_password("alice", "secret")

        assert resp == {"token": "jwt123"}
        assert client._session.headers["Authorization"] == "Bearer jwt123"
        mock_post.assert_called_once_with(
            "http://localhost:8090/api/collections/users/auth-with-password",
            json={"identity": "alice", "password": "secret"},
        )
