import xarray as xr
from qpi_driver.executors.base import Executor

class QbloxExecutor(Executor):
    """Executor subclass for interacting with Qblox instruments and modules."""

    def execute(self, payload: dict) -> xr.Dataset:
        """Execute quantum instructions on Qblox hardware.

        Args:
            payload: Dictionary containing circuit execution options.

        Returns:
            xr.Dataset: The control acquisition dataset.

        Raises:
            NotImplementedError: Always raised since the executor is a placeholder.
        """
        raise NotImplementedError("QbloxExecutor is not implemented yet.")
