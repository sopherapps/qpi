import xarray as xr
from qpi_driver.executors.base import Executor

class QuantifyExecutor(Executor):
    """Executor subclass for interacting with Quantify-scheduler acquisition backends."""

    def execute(self, payload: dict) -> xr.Dataset:
        """Execute quantum instructions using the Quantify scheduler.

        Args:
            payload: Dictionary containing circuit execution options.

        Returns:
            xr.Dataset: The control acquisition dataset.

        Raises:
            NotImplementedError: Always raised since the executor is a placeholder.
        """
        raise NotImplementedError("QuantifyExecutor is not implemented yet.")
