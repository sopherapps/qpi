import time
import random
import numpy as np
import xarray as xr
from qpi_driver.executors.base import Executor

class MockExecutor(Executor):
    def execute(self, payload: dict) -> xr.Dataset:
        n_qubits = payload.get("n_qubits", 2)
        shots = payload.get("shots", 1024)

        # Simulate execution latency
        time.sleep(random.uniform(0.1, 0.5))

        states = [format(i, f"0{n_qubits}b") for i in range(2 ** n_qubits)]
        raw_counts = np.random.multinomial(shots, [1 / len(states)] * len(states))
        
        return xr.Dataset(
            {
                "counts": xr.DataArray(
                    raw_counts.tolist(), 
                    dims=["state"],
                    coords={"state": states}
                ),
                "frequencies": xr.DataArray(
                    (raw_counts / shots).tolist(), 
                    dims=["state"],
                    coords={"state": states}
                )
            },
            attrs={"shots": shots, "n_qubits": n_qubits, "backend": "mock"}
        )
