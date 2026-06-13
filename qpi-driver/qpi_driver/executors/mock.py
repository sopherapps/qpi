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

        # Reconstruct list of shot outcomes
        shot_outcomes = []
        for state_idx, count in enumerate(raw_counts):
            shot_outcomes.extend([states[state_idx]] * count)
        random.shuffle(shot_outcomes)

        data_vars = {}
        for i in range(n_qubits):
            # Qubit state 0 or 1 for each shot. Qubit 0 is LSB, so index is n_qubits - 1 - i.
            qubit_vals = [float(outcome[n_qubits - 1 - i]) for outcome in shot_outcomes]
            data_vars[str(i)] = xr.DataArray(
                [complex(val, 0.0) for val in qubit_vals],
                dims=[f"acq_index_{i}"],
                coords={f"acq_index_{i}": list(range(shots))},
            )

        return xr.Dataset(
            data_vars,
            attrs={"shots": shots, "n_qubits": n_qubits, "backend": self.name},
        )
