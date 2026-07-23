"""Officially maintained drivers that ship with the SDK (RFC 0001 §3, §5).

Drivers are grouped by *operation* — what they do — which is also the
qpi-driver CLI subcommand that runs them. `PROCESS_DRIVERS` are QPUs that run
jobs pushed to them (`process`); `MONITOR_DRIVERS` report upward on a timer
(`monitor`). Each registry maps a device to a run function with a uniform
signature, so a new device is one entry here and needs no CLI changes.
"""

from collections.abc import Callable

from qpi_driver.builtins import bluefors_gen1, qpu
from qpi_driver.builtins.bluefors_gen1 import (
    BlueforsGen1Driver,
)
from qpi_driver.builtins.qpu import QpuDriver, run_qpu_driver

# A driver runner blocks, running one device of an operation until interrupted.
# It takes the universal transport args plus a parsed ``-o`` option dict; see
# qpi_driver.cli for the call site.
DriverRunner = Callable[..., None]

PROCESS_DRIVERS: dict[str, DriverRunner] = {
    device: qpu.run_process for device in qpu.PROCESS_DEVICES
}
MONITOR_DRIVERS: dict[str, DriverRunner] = {
    "bluefors_gen1": bluefors_gen1.run_monitor,
}

__all__ = [
    "QpuDriver",
    "run_qpu_driver",
    "BlueforsGen1Driver",
    "PROCESS_DRIVERS",
    "MONITOR_DRIVERS",
    "DriverRunner",
]
