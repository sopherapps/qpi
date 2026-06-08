import xarray as xr
from qpi_driver.executors.base import Executor

class PrestoExecutor(Executor):
    def execute(self, payload: dict) -> xr.Dataset:
        raise NotImplementedError("PrestoExecutor is not implemented yet.")
