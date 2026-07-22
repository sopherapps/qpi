"""Base SDK for building QPI drivers (RFC 0001 §4, Phase 2).

A driver is an external process that exchanges typed events with QPI-UI: it
handles the events QPI-UI sends it and emits events of its own. Subclass
:class:`QpiDriver`, override :meth:`~QpiDriver.handle_event` to act on each
inbound event (dispatching on its type), and call :meth:`emit` to send an event
upward. :meth:`every` runs a callback on a timer, for drivers that report on
their own schedule rather than in reply to a dispatch.

The base class owns the transport: the inbound NNG PULL socket QPI-UI pushes to,
the outbound NNG PUSH socket the driver emits on, TLS with the pinned root CA,
and the receive loop that decodes each envelope and routes it to its handler. An
event with no matching handler, or whose handler raises, is logged and dropped —
there is no application-level ACK/NACK (RFC 0001 §4).
"""

import logging
import threading
from abc import ABC, abstractmethod
from dataclasses import dataclass
from pathlib import Path
from typing import Callable

import pynng
import requests
from pynng import TLSConfig

from qpi_driver.events import Event

log = logging.getLogger(__name__)

# FIXME: this needs to be user-configurable, with the given default.
#   see how other such configs were handled
RECV_TIMEOUT_MS = 200


@dataclass
class Connection:
    """Transport coordinates a driver connects on, resolved during handshake.

    Attributes:
        host: Hostname or IP of the QPI-UI NNG endpoints.
        in_port: Port the driver PULLs inbound events from (QPI-UI pushes here).
        out_port: Port the driver PUSHes emitted events to (QPI-UI pulls here).
        ca_file: Path to the pinned root CA certificate used for TLS.
    """

    host: str
    in_port: int
    out_port: int
    ca_file: str


class QpiDriver(ABC):
    """Base class for a QPI driver: handles inbound events, emits its own.

    Attributes:
        qpi_addr: Full URL of the QPI-UI server.
        token: The driver's access token; identifies it (and its QPU) to QPI-UI.
        name: Human-readable name for this driver.
        ca_fingerprint: Expected SHA-256 of the server root CA, pinned over TLS.
        ca_file_path: Where the downloaded root CA certificate is written.
    """

    def __init__(
        self,
        qpi_addr: str,
        token: str,
        name: str,
        ca_fingerprint: str = "",
        ca_file_path: str = "./bin/qpi.ca.pem",
    ) -> None:
        self.qpi_addr = qpi_addr
        self.token = token
        self.name = name
        self.ca_fingerprint = ca_fingerprint
        self.ca_file_path = ca_file_path

        self._out_sock: pynng.Push0 | None = None
        self._emit_lock = threading.Lock()
        self._stop = threading.Event()
        self._periodic: list[tuple[float, Callable[[], None]]] = []
        self._threads: list[threading.Thread] = []

    def emit(self, event: Event) -> None:
        """Send an event upward to QPI-UI over the outbound NNG channel.

        Delivery is best-effort, as today: if nothing is listening the event is
        dropped rather than buffered (RFC 0001 §5).

        Raises:
            RuntimeError: If called before the driver has connected.
        """
        if self._out_sock is None:
            raise RuntimeError("cannot emit before the driver is running")

        payload = self._encode_outbound(event)
        with self._emit_lock:
            self._out_sock.send(payload)

    def every(self, interval: float, fn: Callable[[], None]) -> None:
        """Register a callback to run every *interval* seconds while the driver runs.

        Used by drivers that report on their own schedule — e.g. a monitor that
        emits a reading on a timer — independently of any inbound event.
        """
        self._periodic.append((interval, fn))

    def run(self) -> None:
        """Connect to QPI-UI and process events until interrupted.

        Performs the handshake, opens the outbound channel, starts any periodic
        callbacks, then blocks on the inbound receive loop.
        """
        conn = self._connect()

        tls_config = TLSConfig(
            TLSConfig.MODE_CLIENT,
            server_name=conn.host,
            ca_files=conn.ca_file,
        )

        self._out_sock = pynng.Push0(tls_config=tls_config)
        out_addr = f"tls+tcp://{conn.host}:{conn.out_port}"
        self._out_sock.dial(out_addr, block=True)
        log.info("NNG PUSH connected to %s", out_addr)

        self._on_start()
        self._start_periodic()

        try:
            self._recv_loop(conn, tls_config)
        except KeyboardInterrupt:
            log.info("Shutdown signal received")
        finally:
            self._shutdown()

    def _recv_loop(self, conn: Connection, tls_config: TLSConfig) -> None:
        """Pull inbound events and dispatch each to its handler until stopped."""
        in_addr = f"tls+tcp://{conn.host}:{conn.in_port}"
        with pynng.Pull0(tls_config=tls_config, recv_timeout=RECV_TIMEOUT_MS) as sock:
            sock.dial(in_addr, block=True)
            log.info("NNG PULL connected to %s", in_addr)

            while not self._stop.is_set():
                try:
                    raw = sock.recv()
                except pynng.Timeout:
                    continue
                except pynng.Closed:
                    return

                event = self._decode_inbound(raw)
                if event is not None:
                    self._deliver(event)

    # FIXME: update the corresponding RFC to show we use handle_event instead of on_<event>
    @abstractmethod
    def handle_event(self, event: Event) -> None:
        """Act on a single inbound event, dispatching on ``event.type``.

        Implemented per driver. An event a driver does not care about is simply
        ignored; raising signals a rejected event, which is logged and dropped.
        There is no application-level ACK/NACK (RFC 0001 §4).
        """

    def _deliver(self, event: Event) -> None:
        """Pass an event to :meth:`handle_event`, logging and dropping on failure."""
        try:
            self.handle_event(event)
        except Exception:
            log.exception(
                "dropping event %s of type %s: handler failed",
                event.id,
                event.type.value,
            )

    def _start_periodic(self) -> None:
        for interval, fn in self._periodic:
            thread = threading.Thread(
                target=self._run_periodic, args=(interval, fn), daemon=True
            )
            thread.start()
            self._threads.append(thread)

    def _run_periodic(self, interval: float, fn: Callable[[], None]) -> None:
        while not self._stop.wait(interval):
            try:
                fn()
            except Exception:
                log.exception("periodic callback failed")

    def _shutdown(self) -> None:
        log.info("Shutting down driver...")
        self._stop.set()
        self._on_stop()
        if self._out_sock is not None:
            self._out_sock.close()
            self._out_sock = None
        log.info("Shutdown complete.")

    def _connect(self) -> Connection:
        """Handshake with QPI-UI over the shared driver connect endpoint.

        Every driver connects the same way: the token identifies the driver
        (and, transitively, its QPU), and QPI-UI returns the NNG ports and host.
        What differs between drivers is only which events they handle and emit,
        not how they connect (RFC 0001 §3, §8).
        """
        from qpi_driver.driver import _download_root_ca_cert

        resp = requests.post(
            f"{self.qpi_addr}/api/op/drivers/connect",
            json={"token": self.token, "name": self.name},
            timeout=10,
        )
        resp.raise_for_status()
        data = resp.json()

        ca_file = _download_root_ca_cert(
            self.qpi_addr, self.ca_fingerprint, Path(self.ca_file_path)
        )
        return Connection(
            host=data["nng_host"],
            in_port=int(data["nng_in_port"]),
            out_port=int(data["nng_out_port"]),
            ca_file=ca_file,
        )

    def _on_start(self) -> None:
        """Hook run after the outbound channel opens, before the receive loop.

        Subclasses that need background work (e.g. an executor subprocess) start
        it here.
        """

    def _on_stop(self) -> None:
        """Hook run once the receive loop exits, for releasing resources."""

    def _decode_inbound(self, raw: bytes) -> Event | None:
        """Turn a received wire message into an :class:`Event`, or ``None`` to drop it.

        The default parses the shared envelope (RFC 0001 §6); subclasses speaking
        a legacy wire shape override this.
        """
        try:
            return Event.from_json(raw)
        except Exception:
            log.exception("dropping malformed inbound message")
            return None

    def _encode_outbound(self, event: Event) -> bytes:
        """Serialise an outbound event to wire bytes.

        The default emits the shared envelope; subclasses speaking a legacy wire
        shape override this.
        """
        return event.to_json().encode()
