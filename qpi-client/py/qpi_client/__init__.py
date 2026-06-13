"""QPI Client SDK for Python.

A client library for interacting with the QPI quantum computing platform.
Provides both a low-level HTTP client and Qiskit-compatible backend/job classes.

Usage::

    from qpi_client import QPIClient, QPIBackend

    # Low-level client
    client = QPIClient("http://localhost:8090", api_token="my-token")
    job_id = client.submit_job([{"circuit": "OPENQASM 3.0; ..."}])

    # Qiskit integration
    backend = QPIBackend(client, num_qubits=5)
    job = backend.run(my_circuit, shots=1024)
    result = job.result()
"""

from qpi_client.client import QPIClient
from qpi_client.provider import QPIBackend, QPIJob

__all__ = ["QPIClient", "QPIBackend", "QPIJob"]
__version__ = "0.1.0"
