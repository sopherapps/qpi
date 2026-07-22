"""The QPU expressed as a QPI driver (RFC 0001 §4, Phase 2).

A QPU is just the first driver: it handles :attr:`EventType.JOB_DISPATCH` by
running the job on its executor and emits :attr:`EventType.JOB_RESULT` with the
outcome. Execution still happens in a worker subprocess, so a heavy or crashing
executor never blocks or takes down the receive loop, exactly as the standalone
runner does today.

Connection and wire framing come from the base class: the driver connects over
the shared driver endpoint and exchanges the event envelope (RFC 0001 §3, §6).
"""

import logging
import multiprocessing
import threading
from pathlib import Path
from typing import Any

from qpi_driver.driver import _normalize_qpi_addr, _sanitize_name, execute_job
from qpi_driver.events import Event, EventType
from qpi_driver.executors import Executor
from qpi_driver.sdk import QpiDriver

log = logging.getLogger(__name__)


class QpuDriver(QpiDriver):
    """A QPI driver that runs quantum jobs on an executor."""

    def __init__(
        self,
        qpi_addr: str = "http://127.0.0.1:8090",
        token: str = "",
        name: str = "qpu_sim_01",
        executor: str | type[Executor] | Executor = "mock",
        custom_executors: dict[str, type[Executor]] | None = None,
        data_dir: Path = Path("bin/data"),
        ca_fingerprint: str = "",
        ca_file_path: Path = Path("./bin/qpi.ca.pem"),
        **executor_options: Any,
    ) -> None:
        super().__init__(
            qpi_addr=_normalize_qpi_addr(qpi_addr),
            token=token,
            name=_sanitize_name(name),
            ca_fingerprint=ca_fingerprint,
            ca_file_path=Path(ca_file_path).as_posix(),
        )
        self.executor = executor
        self.custom_executors = custom_executors
        self.data_dir = data_dir
        self.executor_options = executor_options

        self._job_queue: multiprocessing.Queue | None = None
        self._result_queue: multiprocessing.Queue | None = None
        self._worker: multiprocessing.Process | None = None
        self._result_pump: threading.Thread | None = None

    def handle_event(self, event: Event) -> None:
        """Run dispatched jobs; ignore everything else (RFC 0001 §8).

        A JobDispatch payload is the job envelope QPI-UI's dispatcher builds —
        ``{job_id, payload}`` — which is exactly what the worker consumes.
        """
        if event.type is EventType.JOB_DISPATCH:
            log.info("Received job %s", event.payload.get("job_id"))
            self._job_queue.put(event.payload)
        else:
            log.warning(
                "dropping event %s: QPU driver does not handle %s",
                event.id,
                event.type.value,
            )

    def _on_start(self) -> None:
        self._job_queue = multiprocessing.Queue()
        self._result_queue = multiprocessing.Queue()

        worker_options = {**self.executor_options, "name": self.name}
        self._worker = multiprocessing.Process(
            target=execute_job,
            kwargs={
                "job_queue": self._job_queue,
                "result_queue": self._result_queue,
                "executor": self.executor,
                "custom_executors": self.custom_executors,
                "data_dir": self.data_dir,
                **worker_options,
            },
            name="QPI-Worker",
            daemon=True,
        )
        self._worker.start()

        self._result_pump = threading.Thread(
            target=self._pump_results, name="QPI-ResultPump", daemon=True
        )
        self._result_pump.start()

    def _pump_results(self) -> None:
        """Drain executor results and emit each as a JobResult event."""
        while True:
            item = self._result_queue.get()
            if item is None:
                log.info("Result pump received shutdown signal")
                return

            job_id = item["job_id"]
            results = {"error": item["error"]} if "error" in item else item["results"]
            log.info("Emitting result for job %s", job_id)
            self.emit(
                Event(
                    type=EventType.JOB_RESULT,
                    driver=self.name,
                    payload={"job_id": job_id, "results": results},
                )
            )

    def _on_stop(self) -> None:
        if self._job_queue is not None:
            _safe_put(self._job_queue, None)
        if self._result_queue is not None:
            _safe_put(self._result_queue, None)

        if self._worker is not None:
            self._worker.join(timeout=2)
            if self._worker.is_alive():
                log.warning("Terminating worker process...")
                self._worker.terminate()
                self._worker.join()


def run_qpu_driver(
    qpi_addr: str = "http://127.0.0.1:8090",
    token: str = "",
    name: str = "qpu_sim_01",
    executor: str | type[Executor] | Executor = "mock",
    custom_executors: dict[str, type[Executor]] | None = None,
    data_dir: Path = Path("bin/data"),
    ca_fingerprint: str = "",
    ca_file_path: Path = Path("./bin/qpi.ca.pem"),
    **executor_options: Any,
) -> None:
    """Run a QPU on the driver framework — the SDK counterpart to ``run_driver``.

    Accepts the same arguments as :func:`qpi_driver.driver.run_driver` so it is a
    drop-in alternative once the framework is enabled.
    """
    QpuDriver(
        qpi_addr=qpi_addr,
        token=token,
        name=name,
        executor=executor,
        custom_executors=custom_executors,
        data_dir=data_dir,
        ca_fingerprint=ca_fingerprint,
        ca_file_path=ca_file_path,
        **executor_options,
    ).run()


def _safe_put(queue: multiprocessing.Queue, item: Any) -> None:
    try:
        queue.put(item)
    except Exception:
        pass
