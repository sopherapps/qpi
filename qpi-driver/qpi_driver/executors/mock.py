import random
import time

import numpy as np
import xarray as xr

from qpi_driver.executors.base import Executor, JobPayload


class MockExecutor(Executor):
    def execute(self, payload: JobPayload) -> xr.Dataset:
        """Execute mock quantum circuit execution by drawing random multinomial samples.

        Args:
            payload: JobPayload specifying n_qubits and shots.

        Returns:
            xr.Dataset: Dataset containing simulated states, counts, and frequencies.
        """
        n_qubits = payload.n_qubits
        shots = payload.shots

        # Simulate execution latency
        time.sleep(random.uniform(0.1, 0.5))

        states = [format(i, f"0{n_qubits}b") for i in range(2**n_qubits)]
        raw_counts = np.random.multinomial(shots, [1 / len(states)] * len(states))

        return xr.Dataset(
            {
                "counts": xr.DataArray(
                    raw_counts.tolist(), dims=["state"], coords={"state": states}
                ),
                "frequencies": xr.DataArray(
                    (raw_counts / shots).tolist(),
                    dims=["state"],
                    coords={"state": states},
                ),
            },
            attrs={"shots": shots, "n_qubits": n_qubits, "backend": "mock"},
        )

