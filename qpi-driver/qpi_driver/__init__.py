import importlib.metadata

try:
    __version__ = importlib.metadata.version("qpi-driver")
except importlib.metadata.PackageNotFoundError:
    __version__ = "0.0.42"

from qpi_driver.driver import run_driver
from qpi_driver.events import Event, EventType
from qpi_driver.executors.base import Executor
from qpi_driver.executors.mock import MockExecutor
from qpi_driver.executors.presto import PrestoExecutor
from qpi_driver.executors.qblox import QbloxExecutor
from qpi_driver.executors.qiskit_aer import QiskitAerExecutor
from qpi_driver.executors.quantify import QuantifyExecutor
from qpi_driver.qpu import QpuDriver, run_qpu_driver
from qpi_driver.sdk import QpiDriver

# FIXME: I expected that qpi-driver would take up a similar folder structure as qpi-client
#   but I don't know if that is in a later phase.

__all__ = [
    "__version__",
    "run_driver",
    "Event",
    "EventType",
    "QpiDriver",
    "QpuDriver",
    "run_qpu_driver",
    "Executor",
    "MockExecutor",
    "QiskitAerExecutor",
    "QuantifyExecutor",
    "QbloxExecutor",
    "PrestoExecutor",
]
