from abc import ABC, abstractmethod

import xarray as xr


class Executor(ABC):
    @abstractmethod
    def execute(self, payload: dict) -> xr.Dataset:
        """Execute the quantum circuit/instructions payload.

        Args:
            payload: Dictionary containing circuit descriptions, qubit count, shots, etc.

        Returns:
            xr.Dataset: Dataset mimicking the raw measurement counts and frequencies.
        """
        pass
