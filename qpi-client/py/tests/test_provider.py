"""Unit tests for qpi_client.provider module."""

from __future__ import annotations

from unittest.mock import MagicMock

import pytest
from qiskit.circuit import QuantumCircuit
from qiskit.providers import JobStatus
from qiskit.result import Result
from qpi_client.client import QPIClient
from qpi_client.provider import QPIBackend, QPIJob


@pytest.fixture
def mock_client() -> MagicMock:
    client = MagicMock(spec=QPIClient)
    client.get_qpu.return_value = {"name": "qpi", "num_qubits": 5}
    client.submit_job.return_value = "job-123"
    client.get_job.return_value = {
        "id": "job-123",
        "status": "completed",
        "results": {
            "circuit_results": [{"counts": {"0x0": 512, "0x3": 512}, "shots": 1024}]
        },
    }
    client.cancel_job.return_value = {"status": "cancelled"}
    return client


@pytest.fixture
def backend(mock_client: MagicMock) -> QPIBackend:
    return QPIBackend(mock_client)


class TestQPIBackend:
    def test_target_property(self, backend: QPIBackend) -> None:
        assert backend.target.num_qubits == 5

    def test_max_circuits(self, backend: QPIBackend) -> None:
        assert backend.max_circuits is None

    def test_run_single_circuit(
        self, backend: QPIBackend, mock_client: MagicMock
    ) -> None:
        qc = QuantumCircuit(2, 2)
        qc.h(0)
        qc.cx(0, 1)
        qc.measure([0, 1], [0, 1])

        job = backend.run(qc, shots=2048)

        assert isinstance(job, QPIJob)
        assert job.job_id() == "job-123"
        mock_client.submit_job.assert_called_once()
        args, kwargs = mock_client.submit_job.call_args
        assert kwargs["shots"] == 2048
        assert len(kwargs["circuits"]) == 1
        assert "circuit" in kwargs["circuits"][0]

    def test_run_multiple_circuits(
        self, backend: QPIBackend, mock_client: MagicMock
    ) -> None:
        qc1 = QuantumCircuit(1, 1)
        qc1.x(0)
        qc1.measure(0, 0)

        qc2 = QuantumCircuit(1, 1)
        qc2.h(0)
        qc2.measure(0, 0)

        job = backend.run([qc1, qc2])

        assert isinstance(job, QPIJob)
        mock_client.submit_job.assert_called_once()
        args, kwargs = mock_client.submit_job.call_args
        assert len(kwargs["circuits"]) == 2

    def test_run_with_parameter_values(
        self, backend: QPIBackend, mock_client: MagicMock
    ) -> None:
        from qiskit.circuit import Parameter

        theta = Parameter("theta")
        qc = QuantumCircuit(1, 1)
        qc.rx(theta, 0)
        qc.measure(0, 0)

        job = backend.run(qc, parameter_values=[{theta: 0.5}])

        assert isinstance(job, QPIJob)
        mock_client.submit_job.assert_called_once()
        args, kwargs = mock_client.submit_job.call_args
        assert len(kwargs["circuits"]) == 1
        # Should have parameter_values in the payload
        assert "parameter_values" in kwargs["circuits"][0]

    def test_default_options(self, backend: QPIBackend) -> None:
        opts = backend._options
        assert opts.get("shots") == 1024
        assert opts.get("meas_level") == 2
        assert opts.get("meas_return") == "single"

    def test_resolve_num_qubits_missing_raises(self, mock_client: MagicMock) -> None:
        mock_client.get_qpu.return_value = {"name": "qpi"}  # no num_qubits key
        with pytest.raises(RuntimeError, match="no valid num_qubits"):
            QPIBackend(mock_client)

    def test_resolve_num_qubits_null_raises(self, mock_client: MagicMock) -> None:
        mock_client.get_qpu.return_value = {"name": "qpi", "num_qubits": None}
        with pytest.raises(RuntimeError, match="no valid num_qubits"):
            QPIBackend(mock_client)

    def test_resolve_num_qubits_api_failure_raises(
        self, mock_client: MagicMock
    ) -> None:
        mock_client.get_qpu.side_effect = RuntimeError("connection refused")
        with pytest.raises(RuntimeError, match="connection refused"):
            QPIBackend(mock_client)

    def test_run_qasm(self, backend: QPIBackend, mock_client: MagicMock) -> None:
        job = backend.run(
            qasm="OPENQASM 3.0; qubit[2] q; bit[2] c; h q[0]; cnot q[0], q[1]; c = measure q;",
            shots=512,
        )

        assert isinstance(job, QPIJob)
        assert job.job_id() == "job-123"
        mock_client.submit_job.assert_called_once()
        args, kwargs = mock_client.submit_job.call_args
        assert kwargs["shots"] == 512
        assert len(kwargs["circuits"]) == 1
        assert (
            kwargs["circuits"][0]["circuit"]
            == "OPENQASM 3.0; qubit[2] q; bit[2] c; h q[0]; cnot q[0], q[1]; c = measure q;"
        )

    def test_run_qasm_with_parameter_values(
        self, backend: QPIBackend, mock_client: MagicMock
    ) -> None:
        job = backend.run(qasm="OPENQASM 3.0; ...", parameter_values=[[0.5, 1.0]])

        args, kwargs = mock_client.submit_job.call_args
        assert kwargs["circuits"][0]["parameter_values"] == [[0.5, 1.0]]

    def test_run_neither_circuit_nor_qasm_raises(self, backend: QPIBackend) -> None:
        with pytest.raises(
            ValueError, match="Either 'circuit' or 'qasm' must be provided"
        ):
            backend.run()

    def test_run_both_circuit_and_qasm_raises(self, backend: QPIBackend) -> None:
        qc = QuantumCircuit(1, 1)
        with pytest.raises(
            ValueError, match="Only one of 'circuit' or 'qasm' should be provided"
        ):
            backend.run(circuit=qc, qasm="OPENQASM 3.0; ...")

    def test_backend_job(self, backend: QPIBackend, mock_client: MagicMock) -> None:
        job = backend.job("job-456")
        assert isinstance(job, QPIJob)
        assert job.job_id() == "job-456"
        assert job.backend() is backend


class TestQPIJob:
    def test_result_completed(self, mock_client: MagicMock) -> None:
        job = QPIJob(
            backend=MagicMock(),
            job_id="job-123",
            client=mock_client,
        )
        result = job.result()

        assert isinstance(result, Result)
        assert result.success is True
        assert len(result.results) == 1
        assert result.job_id == "job-123"

    def test_result_failed_raises(self, mock_client: MagicMock) -> None:
        mock_client.get_job.return_value = {
            "id": "job-123",
            "status": "failed",
            "results": {"error": "simulator error"},
        }
        job = QPIJob(backend=MagicMock(), job_id="job-123", client=mock_client)

        with pytest.raises(RuntimeError, match="simulator error"):
            job.result()

    def test_result_cancelled_raises(self, mock_client: MagicMock) -> None:
        mock_client.get_job.return_value = {
            "id": "job-123",
            "status": "cancelled",
            "results": {},
        }
        job = QPIJob(backend=MagicMock(), job_id="job-123", client=mock_client)

        with pytest.raises(RuntimeError, match="cancelled"):
            job.result()

    def test_status_mapping(self, mock_client: MagicMock) -> None:
        status_map = {
            "pending": JobStatus.QUEUED,
            "queued": JobStatus.QUEUED,
            "running": JobStatus.RUNNING,
            "completed": JobStatus.DONE,
            "failed": JobStatus.ERROR,
            "cancelled": JobStatus.CANCELLED,
        }

        for server_status, expected_status in status_map.items():
            mock_client.get_job.return_value = {
                "id": "job-123",
                "status": server_status,
            }
            job = QPIJob(backend=MagicMock(), job_id="job-123", client=mock_client)
            assert job.status() == expected_status

    def test_cancel(self, mock_client: MagicMock) -> None:
        job = QPIJob(backend=MagicMock(), job_id="job-123", client=mock_client)
        job.cancel()
        mock_client.cancel_job.assert_called_once_with("job-123")

    def test_result_caching(self, mock_client: MagicMock) -> None:
        job = QPIJob(backend=MagicMock(), job_id="job-123", client=mock_client)
        result1 = job.result()
        result2 = job.result()

        # get_job should only be called once because result is cached
        assert mock_client.get_job.call_count == 1
        assert result1 is result2

    def test_result_timeout(self, mock_client: MagicMock) -> None:
        mock_client.get_job.return_value = {
            "id": "job-123",
            "status": "running",
        }
        job = QPIJob(backend=MagicMock(), job_id="job-123", client=mock_client)

        with pytest.raises(TimeoutError):
            job.result(timeout=0.1, wait=0.05)

    def test_build_result_single_circuit(self, mock_client: MagicMock) -> None:
        mock_client.get_job.return_value = {
            "id": "job-123",
            "status": "completed",
            "results": {"counts": {"0x0": 100}, "shots": 100},
        }
        job = QPIJob(backend=MagicMock(), job_id="job-123", client=mock_client)
        result = job.result()

        assert len(result.results) == 1
        assert result.results[0].data.counts == {"0x0": 100}

    def test_build_result_with_memory(self, mock_client: MagicMock) -> None:
        mock_client.get_job.return_value = {
            "id": "job-123",
            "status": "completed",
            "results": {
                "circuit_results": [
                    {"counts": {"0x0": 50}, "memory": ["0x0", "0x0"], "shots": 50}
                ]
            },
        }
        job = QPIJob(backend=MagicMock(), job_id="job-123", client=mock_client)
        result = job.result()

        assert result.results[0].data.memory == ["0x0", "0x0"]
