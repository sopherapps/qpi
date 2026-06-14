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

    def process_result(self, dataset: xr.Dataset, job_id: str) -> dict:
        """Convert raw dataset to Qiskit-compatible result dict.

        Args:
            dataset: The xr.Dataset returned by execute().
            job_id: The unique ID of the quantum job.

        Returns:
            dict: Qiskit-compatible result dict.

        Raises:
            NotImplementedError: Always raised since the executor is a placeholder.
        """
        raise NotImplementedError("QbloxExecutor is not implemented yet.")
