"""
QPi Hardware Driver
===================
Thread architecture (all in one process)
-----------------------------------------
  Main Thread   : HTTP handshake
  NNG Puller    : NNG PULL (receives jobs from Go dispatcher)
                  → feeds job_queue
  Worker        : pulls from job_queue → executes quantum circuit
                  → translates xarray→Qiskit → feeds result_queue
  Result Sender : pulls from result_queue → NNG PUSH (sends back to Go)

Environment variables
---------------------
  GO_SERVER_HOST        LAN IP / hostname of the Go PocketBase server (default: 127.0.0.1)
  GO_SERVER_PORT        PocketBase HTTP port (default: 8090)
  REGISTRATION_TOKEN    Token that matches a qpus.registration_token record
  QPU_NAME              Human-readable name for this QPU (default: "QPU-Sim-01")
  DRIVER_BACKEND        Which backend to use: "mock" | "qiskit_aer"  (default: mock)
"""

from __future__ import annotations

import json
import logging
import os
import queue
import random
import threading
import time

import numpy as np
import xarray as xr

import pynng
import requests

from qiskit.result import Result
from qiskit.result.models import ExperimentResult, ExperimentResultData

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

GO_SERVER_HOST     = os.getenv("GO_SERVER_HOST", "127.0.0.1")
GO_SERVER_PORT     = int(os.getenv("GO_SERVER_PORT", "8090"))
REGISTRATION_TOKEN = os.getenv("REGISTRATION_TOKEN", "")
QPU_NAME           = os.getenv("QPU_NAME", "QPU-Sim-01")
DRIVER_BACKEND     = os.getenv("DRIVER_BACKEND", "mock")  # "mock" | "qiskit_aer"

REGISTER_URL       = f"http://{GO_SERVER_HOST}:{GO_SERVER_PORT}/api/qpu/register"

logging.basicConfig(
    level=logging.INFO,
    format="[%(threadName)-16s] %(levelname)s %(message)s",
)
log = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# HTTP handshake
# ---------------------------------------------------------------------------

def do_handshake() -> dict:
    """POST to /api/qpu/register and return {nng_command_port, nng_result_port, auth_token}."""
    if not REGISTRATION_TOKEN:
        raise ValueError("REGISTRATION_TOKEN env var must be set")

    payload = {"name": QPU_NAME, "registration_token": REGISTRATION_TOKEN}
    log.info("Handshaking with %s …", REGISTER_URL)
    resp = requests.post(REGISTER_URL, json=payload, timeout=10)
    resp.raise_for_status()
    data = resp.json()
    log.info("Handshake OK — cmd_port=%s  res_port=%s",
             data["nng_command_port"], data["nng_result_port"])
    return data

# ---------------------------------------------------------------------------
# Thread 1 — NNG PULL (receives jobs)
# ---------------------------------------------------------------------------

def nng_puller(cmd_port: int, job_queue: queue.Queue) -> None:
    """PULL jobs from Go dispatcher → push to job_queue."""
    addr = f"tcp://{GO_SERVER_HOST}:{cmd_port}"
    log.info("Connecting NNG PULL → %s", addr)
    with pynng.Pull0() as sock:
        sock.dial(addr, block=True)
        log.info("NNG PULL connected to %s", addr)
        while True:
            try:
                raw = sock.recv()
                job = json.loads(raw)
                log.info("Received job %s", job.get("job_id"))
                job_queue.put(job)
            except Exception as exc:
                log.error("PULL error: %s", exc)
                time.sleep(1)

# ---------------------------------------------------------------------------
# Quantum execution helpers
# ---------------------------------------------------------------------------

def execute_circuit(job: dict) -> xr.Dataset:
    """
    Simulate quantum circuit execution.
    Returns an xarray Dataset mimicking quantify-scheduler raw measurement output.
    """
    payload = job.get("payload", {})
    # payload may be a JSON string from PocketBase
    if isinstance(payload, str):
        try:
            payload = json.loads(payload)
        except (json.JSONDecodeError, TypeError):
            payload = {}
    n_qubits = payload.get("n_qubits", 2) if isinstance(payload, dict) else 2
    shots    = payload.get("shots", 1024) if isinstance(payload, dict) else 1024

    if DRIVER_BACKEND == "qiskit_aer":
        try:
            from qiskit_aer import AerSimulator  # type: ignore
            from qiskit import QuantumCircuit, transpile
            qc = QuantumCircuit(n_qubits, n_qubits)
            qc.h(0)
            if n_qubits > 1:
                qc.cx(0, 1)
            qc.measure_all()
            simulator = AerSimulator()
            t_qc = transpile(qc, simulator)
            sim_result = simulator.run(t_qc, shots=shots).result()
            counts = sim_result.get_counts(t_qc)
            states = list(counts.keys())
            freqs  = [counts[s] / shots for s in states]
            return xr.Dataset(
                {"counts": xr.DataArray(list(counts.values()), dims=["state"],
                                        coords={"state": states}),
                 "frequencies": xr.DataArray(freqs, dims=["state"],
                                             coords={"state": states})},
                attrs={"shots": shots, "n_qubits": n_qubits, "backend": "qiskit_aer"},
            )
        except ImportError:
            log.warning("qiskit-aer not installed, falling back to mock")

    # Pure mock — random binary counts
    time.sleep(random.uniform(0.1, 0.5))  # simulate execution time
    states     = [format(i, f"0{n_qubits}b") for i in range(2 ** n_qubits)]
    raw_counts = np.random.multinomial(shots, [1 / len(states)] * len(states))
    return xr.Dataset(
        {"counts":      xr.DataArray(raw_counts.tolist(), dims=["state"],
                                     coords={"state": states}),
         "frequencies": xr.DataArray((raw_counts / shots).tolist(), dims=["state"],
                                     coords={"state": states})},
        attrs={"shots": shots, "n_qubits": n_qubits, "backend": "mock"},
    )


def xarray_to_qiskit_counts(dataset: xr.Dataset) -> dict:
    """
    Convert an xarray.Dataset (from mock / quantify-scheduler) to a
    Qiskit-compatible Result-like dict with memory counts.
    """
    counts_da = dataset.get("counts")
    if counts_da is None:
        return {"raw": str(dataset)}

    states = counts_da.coords["state"].values.tolist()
    counts = counts_da.values.tolist()
    counts_dict = {s: int(c) for s, c in zip(states, counts)}
    shots  = int(dataset.attrs.get("shots", sum(counts)))
    backend = dataset.attrs.get("backend", "mock")

    # Build a minimal Qiskit Result object and serialise
    exp_data = ExperimentResultData(counts={hex(int(s, 2)): c
                                            for s, c in counts_dict.items()})
    exp_result = ExperimentResult(
        shots=shots,
        success=True,
        data=exp_data,
        status="DONE",
        name="qpi_job",
    )
    result = Result(
        backend_name=f"qpi_{backend}",
        backend_version="1.0.0",
        qobj_id="qpi",
        job_id="qpi_job",
        success=True,
        results=[exp_result],
    )

    return {
        "counts":           counts_dict,
        "hex_counts":       result.get_counts(0),
        "shots":            shots,
        "backend":          backend,
        "success":          True,
    }

# ---------------------------------------------------------------------------
# Thread 2 — Worker (execute + translate)
# ---------------------------------------------------------------------------

def worker(job_queue: queue.Queue, result_queue: queue.Queue) -> None:
    """Consume job_queue → execute → translate to dict → push to result_queue."""
    log.info("Worker started")
    while True:
        job = job_queue.get()
        job_id = job.get("job_id", "unknown")
        log.info("Executing job %s", job_id)
        try:
            dataset = execute_circuit(job)
            qiskit_data = xarray_to_qiskit_counts(dataset)
            result_queue.put({"job_id": job_id, "results": qiskit_data})
            log.info("Job %s executed + translated", job_id)
        except Exception as exc:
            log.error("Job %s failed: %s", job_id, exc)
            result_queue.put({"job_id": job_id, "results": {"error": str(exc)}})

# ---------------------------------------------------------------------------
# Thread 3 — Result sender (NNG PUSH)
# ---------------------------------------------------------------------------

def result_sender(res_port: int, result_queue: queue.Queue) -> None:
    """Consume result_queue → NNG PUSH back to Go."""
    addr = f"tcp://{GO_SERVER_HOST}:{res_port}"
    log.info("Connecting NNG PUSH → %s", addr)
    with pynng.Push0() as sock:
        sock.dial(addr, block=True)
        log.info("NNG PUSH connected to %s", addr)
        while True:
            item = result_queue.get()
            job_id = item["job_id"]
            try:
                msg = json.dumps(item)
                sock.send(msg.encode())
                log.info("Sent results for job %s", job_id)
            except Exception as exc:
                log.error("Failed to send results for job %s: %s", job_id, exc)

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    info = do_handshake()
    cmd_port = info["nng_command_port"]
    res_port = info["nng_result_port"]

    job_queue:    queue.Queue = queue.Queue()
    result_queue: queue.Queue = queue.Queue()

    # Thread 2: worker
    t_worker = threading.Thread(
        target=worker,
        args=(job_queue, result_queue),
        name="Worker",
        daemon=True,
    )
    t_worker.start()

    # Thread 3: result sender
    t_sender = threading.Thread(
        target=result_sender,
        args=(res_port, result_queue),
        name="ResultSender",
        daemon=True,
    )
    t_sender.start()

    # Thread 1 (main → blocks): NNG puller
    try:
        nng_puller(cmd_port, job_queue)
    except KeyboardInterrupt:
        log.info("Shutting down driver")


if __name__ == "__main__":
    main()
