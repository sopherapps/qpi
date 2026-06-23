import importlib.metadata

try:
    __version__ = importlib.metadata.version("qpi-driver")
except importlib.metadata.PackageNotFoundError:
    __version__ = "0.0.20"

from qpi_driver.driver import run_driver
from qpi_driver.executors.base import Executor
from qpi_driver.executors.mock import MockExecutor
from qpi_driver.executors.presto import PrestoExecutor
from qpi_driver.executors.qblox import QbloxExecutor
from qpi_driver.executors.qiskit_aer import QiskitAerExecutor
from qpi_driver.executors.quantify import QuantifyExecutor

__all__ = [
    "__version__",
    "run_driver",
    "Executor",
    "MockExecutor",
    "QiskitAerExecutor",
    "QuantifyExecutor",
    "QbloxExecutor",
    "PrestoExecutor",
]
