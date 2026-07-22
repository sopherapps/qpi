"""A minimal stand-in for a Bluefors Control API server, used only by the e2e
suite to exercise the cryostat monitor driver without real cryostat hardware
(RFC 0001 §7, Phase 3).

It serves GET requests under /values/<path> with the same response shape the
real Bluefors Remote Access Control API Gen. 1 returns for a value-tree node
(Technical Reference, Appendix I: "values"), with a value that changes on
every request so the monitor's polling and the dashboard's live chart have
something to show.
"""

import itertools
import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class _Handler(BaseHTTPRequestHandler):
    # One shared counter is enough to give every channel a moving value; the
    # mock does not need to distinguish channels to be useful for the e2e
    # smoke test.
    _counter = itertools.count()

    def do_GET(self):  # noqa: N802 - required name by http.server
        if not self.path.startswith("/values/"):
            self.send_response(404)
            self.end_headers()
            return

        path = self.path.split("?", 1)[0]
        channel = path[len("/values/") :].replace("/", ".")
        value = 0.010 + next(self._counter) * 0.001

        body = json.dumps(
            {
                "data": {
                    "name": channel,
                    "type": "Value.Number.Float.Unit",
                    "content": {
                        "read_only": True,
                        "maximum_age": 5000,
                        "lockable": False,
                        "owner": "driver.mock",
                        "latest_valid_value": {
                            "value": f"{value:.4f}",
                            "outdated": False,
                            "date": 0,
                            "status": "SYNCHRONIZED",
                            "exception": "",
                        },
                        "latest_value": {
                            "value": f"{value:.4f}",
                            "outdated": False,
                            "date": 0,
                            "status": "SYNCHRONIZED",
                            "exception": "",
                        },
                    },
                }
            }
        ).encode()

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):  # silence per-request access logs
        pass


def start(port: int = 0) -> tuple[ThreadingHTTPServer, threading.Thread]:
    """Start the mock server on a background thread and return it plus its port.

    Pass ``port=0`` to let the OS assign a free port, then read
    ``server.server_address[1]``.
    """
    server = ThreadingHTTPServer(("127.0.0.1", port), _Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def stop(server: ThreadingHTTPServer) -> None:
    server.shutdown()
    server.server_close()
