import importlib.metadata

try:
    __version__ = importlib.metadata.version("qpi-driver")
except importlib.metadata.PackageNotFoundError:
    __version__ = "0.0.42"

from qpi_driver.builtins.bluefors_gen1 import (
    BlueforsGen1Driver,
)
from qpi_driver.builtins.qpu import QpuDriver, run_qpu_driver
from qpi_driver.driver import run_driver
from qpi_driver.events import Event, EventType
from qpi_driver.executors.base import Executor
from qpi_driver.executors.mock import MockExecutor
from qpi_driver.executors.presto import PrestoExecutor
from qpi_driver.executors.qblox import QbloxExecutor
from qpi_driver.executors.qiskit_aer import QiskitAerExecutor
from qpi_driver.executors.quantify import QuantifyExecutor
from qpi_driver.sdk import QpiDriver

# qpi-driver is intentionally still a single Python package today, unlike
# qpi-client's per-language `go/`, `js/`, `py/` layout — there is only one
# driver SDK language so far. Growing qpi-driver into that same per-language
# layout (moving this package to `qpi-driver/py/qpi_driver/` and adding
# `qpi-driver/go/`, `qpi-driver/js/`) is planned in ROADMAP.md Phase 4 ("More
# language SDKs"), once TypeScript and Go SDKs actually exist to justify it.

__all__ = [
    "__version__",
    "run_driver",
    "Event",
    "EventType",
    "QpiDriver",
    "QpuDriver",
    "run_qpu_driver",
    "BlueforsGen1Driver",
    "Executor",
    "MockExecutor",
    "QiskitAerExecutor",
    "QuantifyExecutor",
    "QbloxExecutor",
    "PrestoExecutor",
]
