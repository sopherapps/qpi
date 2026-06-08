from abc import ABC, abstractmethod
import xarray as xr

class Executor(ABC):
    @abstractmethod
    def execute(self, payload: dict) -> xr.Dataset:
        """
        Execute the quantum circuit/instructions payload.
        Returns an xarray.Dataset mimicking the raw measurement outputs.
        """
        pass
