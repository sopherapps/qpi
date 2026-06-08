import xarray as xr
from qpi_driver.executors.base import Executor

class QuantifyExecutor(Executor):
    def execute(self, payload: dict) -> xr.Dataset:
        raise NotImplementedError("QuantifyExecutor is not implemented yet.")
