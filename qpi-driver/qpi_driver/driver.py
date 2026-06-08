from typing import Any
from dataclasses import dataclass
import json
import logging
import multiprocessing
import os
import time
import requests
import pynng

from qpi_driver.executors import Executor

# Setup logging for the main process.
# We stick with the standard Python logging library instead of introducing logurus
# to avoid bloating the package with external dependencies and to keep integration simple.
logging.basicConfig(
    level=logging.INFO,
    format="[MainProcess] %(levelname)s %(message)s",
)
log = logging.getLogger(__name__)


@dataclass
class HandshakeInfo:
    """Strongly typed handshake response containing assigned port configurations and credentials.

    Attributes:
        nng_command_port: Port allocated for NNG command dispatch.
        nng_result_port: Port allocated for NNG result collection.
        auth_token: Static authorization token assigned to the QPU.
    """
    nng_command_port: int
    nng_result_port: int
    auth_token: str


def do_handshake(host: str, port: int, token: str, name: str) -> HandshakeInfo:
    """POST to /api/qpu/register and return dynamic port configurations.

    Args:
        host: Hostname or IP of the Go PocketBase server.
        port: PocketBase HTTP port.
        token: Unique QPU registration token.
        name: Human-readable name for this QPU.

    Returns:
        HandshakeInfo: Strongly typed port and token credentials.

    Raises:
        ValueError: If the registration token is empty.
        requests.RequestException: If the HTTP request fails.
    """
    if not token:
        raise ValueError("Registration token must be provided")

    register_url = f"http://{host}:{port}/api/qpu/register"
    payload = {"name": name, "registration_token": token}
    log.info("Handshaking with %s …", register_url)
    resp = requests.post(register_url, json=payload, timeout=10)
    resp.raise_for_status()
    data = resp.json()
    log.info("Handshake OK — cmd_port=%s  res_port=%s",
             data["nng_command_port"], data["nng_result_port"])
    return HandshakeInfo(
        nng_command_port=int(data["nng_command_port"]),
        nng_result_port=int(data["nng_result_port"]),
        auth_token=data.get("auth_token", "")
    )


def execute_job(
    job_queue: multiprocessing.Queue,
    result_queue: multiprocessing.Queue,
    executor: str | type[Executor] | Executor,
    custom_executors: dict[str, type[Executor]] | None,
    data_dir: str,
    **executor_options: Any
) -> None:
    """
    Worker process: pulls job dicts from job_queue, executes them using the resolved
    executor, saves the resulting xarray.Dataset as a NetCDF file, and pushes the file path
    to result_queue.

    Args:
        job_queue: multiprocessing.Queue for receiving job dicts
        result_queue: multiprocessing.Queue for sending results (file paths or errors)
        executor: Executor specification (string key, class, or instance)
        custom_executors: Optional dict of custom executors for resolving string keys
        data_dir: Directory where NetCDF files should be saved
        executor_options: additional kwargs to pass when instantiating the executor
    """
    # Override logging config inside worker process
    logging.basicConfig(
        level=logging.INFO,
        format="[WorkerProcess] %(levelname)s %(message)s",
    )
    w_log = logging.getLogger("worker")
    w_log.info("Worker process started")

    from qpi_driver.executors import resolve_executor

    try:
        executor_instance = resolve_executor(executor, custom_executors, **executor_options)
    except Exception as exc:
        w_log.error("Failed to resolve executor: %s", exc)
        result_queue.put({"job_id": "init_error", "error": f"Failed to resolve executor: {exc}"})
        return

    os.makedirs(data_dir, exist_ok=True)

    while True:
        try:
            job = job_queue.get()
            if job is None:  # Poison pill
                w_log.info("Worker process received shutdown signal")
                break

            job_id = job.get("job_id", "unknown")
            w_log.info("Worker process executing job %s", job_id)

            try:
                payload = job.get("payload", {})
                if isinstance(payload, str):
                    try:
                        payload = json.loads(payload)
                    except Exception:
                        payload = {}

                dataset = executor_instance.execute(payload)

                # Save dataset to a NetCDF file
                filepath = os.path.join(data_dir, f"job_{job_id}.nc")
                dataset.to_netcdf(filepath, engine="scipy")

                result_queue.put({"job_id": job_id, "filepath": filepath})
                w_log.info("Worker process completed job %s, saved to %s", job_id, filepath)
            except Exception as exc:
                w_log.error("Worker process failed job %s: %s", job_id, exc)
                result_queue.put({"job_id": job_id, "error": str(exc)})
        except KeyboardInterrupt:
            break
        except Exception as exc:
            w_log.error("Worker loop exception: %s", exc)


def process_results(
    result_queue: multiprocessing.Queue,
    res_port: int,
    host: str
) -> None:
    """Translator process: pulls results from result_queue, loads xarray.Dataset from disk,

    translates it to Qiskit Results format, deletes the NetCDF file, and pushes the JSON string
    back to the Go orchestrator via NNG PUSH.

    Args:
        result_queue: Queue used to receive filepaths or error dicts from the worker.
        res_port: Port allocated for the NNG PUSH socket to return results.
        host: Hostname or IP of the Go PocketBase server.
    """
    # Override logging config inside translator process
    logging.basicConfig(
        level=logging.INFO,
        format="[TranslatorProcess] %(levelname)s %(message)s",
    )
    t_log = logging.getLogger("translator")
    t_log.info("Translator process started")

	# Import dependencies locally
    import xarray as xr
    from qiskit.result import Result
    from qiskit.result.models import ExperimentResult, ExperimentResultData

    def xarray_to_qiskit_counts(dataset: xr.Dataset, job_id: str) -> dict:
        """Convert xarray.Dataset measurements to a Qiskit compatible result dict.

        Args:
            dataset: The xarray.Dataset containing quantum measurement counts and metadata.
            job_id: The unique ID of the quantum job.

        Returns:
            dict: Qiskit results format mapped to hex counts, shots, and status keys.
        """
        counts_da = dataset.get("counts")
        if counts_da is None:
            return {"raw": str(dataset)}

        states = counts_da.coords["state"].values.tolist()
        counts = counts_da.values.tolist()
        counts_dict = {s: int(c) for s, c in zip(states, counts)}
        shots = int(dataset.attrs.get("shots", sum(counts)))
        backend = dataset.attrs.get("backend", "mock")

        # Build a minimal Qiskit Result object and serialise
        expt_data = ExperimentResultData(
            counts={hex(int(s, 2)): c for s, c in counts_dict.items()}
        )
        exp_result = ExperimentResult(
            shots=shots,
            success=True,
            data=expt_data,
            status="DONE",
            name="qpi_job",
        )
        # We omit the arbitrary 'qpi_' prefix from the backend name.
        # We dynamically assign job_id and qobj_id from the processed job.
        result = Result(
            backend_name=backend,
            backend_version="1.0.0",
            qobj_id=job_id,
            job_id=job_id,
            success=True,
            results=[exp_result],
        )

        return {
            "counts": counts_dict,
            "hex_counts": result.get_counts(0),
            "shots": shots,
            "backend": backend,
            "success": True,
        }

    addr = f"tcp://{host}:{res_port}"
    t_log.info("Connecting NNG PUSH → %s", addr)

    with pynng.Push0() as sock:
        sock.dial(addr, block=True)
        t_log.info("NNG PUSH connected to %s", addr)

        while True:
            try:
                item = result_queue.get()
                if item is None:  # Poison pill
                    t_log.info("Translator process received shutdown signal")
                    break

                job_id = item["job_id"]

                if "error" in item:
                    # Report execution error
                    msg_dict = {"job_id": job_id, "results": {"error": item["error"]}}
                    t_log.info("Sending error results for job %s", job_id)
                else:
                    filepath = item["filepath"]
                    t_log.info("Translator process loading dataset from %s", filepath)
                    try:
                        # Load dataset into memory
                        dataset = xr.load_dataset(filepath, engine="scipy")

                        # Delete file immediately
                        try:
                            os.remove(filepath)
                        except Exception as e:
                            t_log.warning("Could not delete file %s: %s", filepath, e)

                        qiskit_data = xarray_to_qiskit_counts(dataset, job_id)
                        msg_dict = {"job_id": job_id, "results": qiskit_data}
                        t_log.info("Sending results for job %s", job_id)
                    except Exception as exc:
                        t_log.error("Failed to load/translate dataset for job %s: %s", job_id, exc)
                        msg_dict = {"job_id": job_id, "results": {"error": str(exc)}}
                        if os.path.exists(filepath):
                            try:
                                os.remove(filepath)
                            except Exception:
                                pass

                msg = json.dumps(msg_dict)
                sock.send(msg.encode())

            except KeyboardInterrupt:
                break
            except Exception as exc:
                t_log.error("Translator process exception: %s", exc)


def run_driver(
    host: str,
    port: int,
    token: str,
    name: str,
    executor: str | type[Executor] | Executor = "mock",
    custom_executors: dict[str, type[Executor]] | None = None,
    data_dir: str = "bin/data"
) -> None:
    """Run the QPI Python hardware driver.

    Args:
        host: Hostname or IP of the Go PocketBase server.
        port: PocketBase HTTP port.
        token: Unique QPU registration token.
        name: Human-readable name for this QPU.
        executor: Executor specification (string key, class, or instance).
        custom_executors: Optional dict of custom executors for resolving string keys.
        data_dir: Directory where NetCDF files should be saved.
    """
    # do_handshake returns strongly typed dataclass
    info = do_handshake(host, port, token, name)
    cmd_port = info.nng_command_port
    res_port = info.nng_result_port

    # Create queues
    job_queue = multiprocessing.Queue()
    result_queue = multiprocessing.Queue()

    # 2. Start Worker Process
    worker = multiprocessing.Process(
        target=execute_job,
        args=(job_queue, result_queue, executor, custom_executors, data_dir),
        name="QPI-Worker",
        daemon=True
    )
    worker.start()

    # 3. Start Translator Process
    translator = multiprocessing.Process(
        target=process_results,
        args=(result_queue, res_port, host),
        name="QPI-Translator",
        daemon=True
    )
    translator.start()

    # 4. Start NNG PULL (commands) in Main Process
    addr = f"tcp://{host}:{cmd_port}"
    log.info("Connecting NNG PULL → %s", addr)

    try:
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
                    # Retry with 0.2s delay for faster reconnection times
                    time.sleep(0.2)
    except KeyboardInterrupt:
        log.info("Shutdown signal received")
    finally:
        log.info("Shutting down worker and translator processes...")
        # Note: We use poison pills (None sentinel values) to cleanly and instantly unblock
        # queue get() calls in the worker and translator processes without wasteful CPU polling.
        try:
            job_queue.put(None)
        except Exception:
            pass
        try:
            result_queue.put(None)
        except Exception:
            pass

        # Give processes time to shut down cleanly, or terminate them
        worker.join(timeout=2)
        if worker.is_alive():
            log.warning("Terminating worker process...")
            worker.terminate()
            worker.join()

        translator.join(timeout=2)
        if translator.is_alive():
            log.warning("Terminating translator process...")
            translator.terminate()
            translator.join()

        log.info("Shutdown complete.")
