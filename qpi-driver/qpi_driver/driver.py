import json
import logging
import multiprocessing
import os
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import pynng
import requests

from qpi_driver.executors import Executor, JobPayload

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


def do_handshake(
    host: str,
    port: int,
    token: str,
    name: str,
    executor_type: str = "",
    device_config: dict[str, Any] | None = None,
) -> HandshakeInfo:
    """POST to /api/qpu/register and return dynamic port configurations.

    Args:
        host: Hostname or IP of the Go PocketBase server.
        port: PocketBase HTTP port.
        token: Unique QPU registration token.
        name: Human-readable name for this QPU.
        executor_type: The executor backend type (e.g. ``"mock"``, ``"qiskit_aer"``).
        device_config: Optional device configuration dict to store on the QPU record.

    Returns:
        HandshakeInfo: Strongly typed port and token credentials.

    Raises:
        ValueError: If the registration token is empty.
        requests.RequestException: If the HTTP request fails.
    """
    if not token:
        raise ValueError("Registration token must be provided")

    register_url = f"http://{host}:{port}/api/qpu/register"
    payload: dict[str, Any] = {"name": name, "registration_token": token}
    if executor_type:
        payload["executor_type"] = executor_type
    if device_config:
        payload["device_config"] = device_config

    log.info("Handshaking with %s …", register_url)
    resp = requests.post(register_url, json=payload, timeout=10)
    resp.raise_for_status()
    data = resp.json()
    log.info(
        "Handshake OK — cmd_port=%s  res_port=%s",
        data["nng_command_port"],
        data["nng_result_port"],
    )
    return HandshakeInfo(
        nng_command_port=int(data["nng_command_port"]),
        nng_result_port=int(data["nng_result_port"]),
        auth_token=data.get("auth_token", ""),
    )


def execute_job(
    job_queue: multiprocessing.Queue,
    result_queue: multiprocessing.Queue,
    executor: str | type[Executor] | Executor,
    custom_executors: dict[str, type[Executor]] | None,
    data_dir: Path,
    **executor_options: Any,
) -> None:
    """
    Worker process: pulls job dicts from job_queue, executes them using the resolved
    executor, converts results to Qiskit-format dicts via ``executor.process_result()``,
    and pushes the result dicts to result_queue.

    Args:
        job_queue: multiprocessing.Queue for receiving job dicts
        result_queue: multiprocessing.Queue for sending result dicts or errors
        executor: Executor specification (string key, class, or instance)
        custom_executors: Optional dict of custom executors for resolving string keys
        data_dir: Directory for executor working data
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
        options = executor_options.copy()
        if "data_dir" not in options:
            options["data_dir"] = data_dir
        executor_instance = resolve_executor(executor, custom_executors, **options)
    except Exception as exc:
        w_log.error("Failed to resolve executor: %s", exc)
        result_queue.put(
            {"job_id": "init_error", "error": f"Failed to resolve executor: {exc}"}
        )
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
                payload_dict = job.get("payload", {})
                if isinstance(payload_dict, str):
                    try:
                        payload_dict = json.loads(payload_dict)
                    except Exception:
                        payload_dict = {}

                payload_dict.update(dict(id=job_id))
                payload = JobPayload.from_dict(payload_dict)
                dataset = executor_instance.execute(payload)
                result_dict = executor_instance.process_result(dataset, job_id)
                result_queue.put({"job_id": job_id, "results": result_dict})
                w_log.info("Worker process completed job %s", job_id)
            except Exception as exc:
                w_log.error("Worker process failed job %s: %s", job_id, exc)
                result_queue.put({"job_id": job_id, "error": str(exc)})
        except KeyboardInterrupt:
            break
        except Exception as exc:
            w_log.error("Worker loop exception: %s", exc)

    # release resources
    executor_instance.close()


def send_results(result_queue: multiprocessing.Queue, res_port: int, host: str) -> None:
    """Result sender process: reads Qiskit-format result dicts from result_queue
    and pushes them to the Go orchestrator via NNG PUSH.

    Args:
        result_queue: Queue used to receive result dicts from the worker.
        res_port: Port allocated for the NNG PUSH socket to return results.
        host: Hostname or IP of the Go PocketBase server.
    """
    logging.basicConfig(
        level=logging.INFO,
        format="[ResultSender] %(levelname)s %(message)s",
    )
    rs_log = logging.getLogger("result_sender")
    rs_log.info("Result sender process started")

    addr = f"tcp://{host}:{res_port}"
    rs_log.info("Connecting NNG PUSH → %s", addr)

    with pynng.Push0() as sock:
        sock.dial(addr, block=True)
        rs_log.info("NNG PUSH connected to %s", addr)

        while True:
            try:
                item = result_queue.get()
                if item is None:  # Poison pill
                    rs_log.info("Result sender process received shutdown signal")
                    break

                job_id = item["job_id"]

                if "error" in item:
                    msg_dict = {"job_id": job_id, "results": {"error": item["error"]}}
                    rs_log.info("Sending error results for job %s", job_id)
                else:
                    msg_dict = {"job_id": job_id, "results": item["results"]}
                    rs_log.info("Sending results for job %s", job_id)

                msg = json.dumps(msg_dict)
                sock.send(msg.encode())

            except KeyboardInterrupt:
                break
            except Exception as exc:
                rs_log.error("Result sender process exception: %s", exc)


def run_driver(
    host: str,
    port: int,
    token: str,
    name: str,
    executor: str | type[Executor] | Executor = "mock",
    custom_executors: dict[str, type[Executor]] | None = None,
    data_dir: Path = Path("bin/data"),
    **executor_options: Any,
) -> None:
    """Run the QPI Python hardware driver.

    Args:
        host: Hostname or IP of the Go PocketBase server.
        port: PocketBase HTTP port.
        token: Unique QPU registration token.
        name: Human-readable name for this QPU.
        executor: Executor specification (string key, class, or instance).
        custom_executors: Optional dict of custom executors for resolving string keys.
        data_dir: Directory for executor working data.
        executor_options: other options to pass to the executor.
    """
    # Determine executor type string for registration
    executor_type_str = ""
    if isinstance(executor, str):
        executor_type_str = executor
    elif hasattr(executor, "name"):
        executor_type_str = executor.name
    elif hasattr(executor, "__name__"):
        executor_type_str = executor.__name__

    # Extract device config from executor options if present
    device_config = executor_options.get("device_config")
    if device_config is None:
        # Try to build a minimal config from known options
        cfg: dict[str, Any] = {}
        for key in ("quantify_hardware_config", "quantify_device_config", "is_dummy"):
            if key in executor_options:
                val = executor_options[key]
                if hasattr(val, "__fspath__"):
                    cfg[key] = str(val)
                else:
                    cfg[key] = val
        if cfg:
            device_config = cfg

    # do_handshake returns strongly typed dataclass
    info = do_handshake(
        host,
        port,
        token,
        name,
        executor_type=executor_type_str,
        device_config=device_config,
    )
    cmd_port = info.nng_command_port
    res_port = info.nng_result_port

    # Create queues
    job_queue = multiprocessing.Queue()
    result_queue = multiprocessing.Queue()

    # add extra args to be passed to the executor
    executor_options.update(dict(name=name))

    # 2. Start Worker Process
    worker = multiprocessing.Process(
        target=execute_job,
        args=(job_queue, result_queue, executor, custom_executors, data_dir),
        kwargs=executor_options,
        name="QPI-Worker",
        daemon=True,
    )
    worker.start()

    # 3. Start Result Sender Process
    result_sender = multiprocessing.Process(
        target=send_results,
        args=(result_queue, res_port, host),
        name="QPI-ResultSender",
        daemon=True,
    )
    result_sender.start()

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
        log.info("Shutting down worker and result sender processes...")
        # Note: We use poison pills (None sentinel values) to cleanly and instantly unblock
        # queue get() calls in the worker and result sender processes without wasteful CPU polling.
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

        result_sender.join(timeout=2)
        if result_sender.is_alive():
            log.warning("Terminating result sender process...")
            result_sender.terminate()
            result_sender.join()

        log.info("Shutdown complete.")
