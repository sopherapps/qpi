import xarray as xr
from qpi_driver.executors.base import Executor

class QbloxExecutor(Executor):
    def execute(self, payload: dict) -> xr.Dataset:
        raise NotImplementedError("QbloxExecutor is not implemented yet.")
