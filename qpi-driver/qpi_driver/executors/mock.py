import random
import time

import numpy as np
import xarray as xr

from qpi_driver.executors.base import CircuitPayload, Executor, JobPayload


class MockExecutor(Executor):
    def execute(self, payload: JobPayload) -> xr.Dataset:
        """Execute mock quantum circuit execution by drawing random multinomial samples.

        For multi-circuit payloads, results are concatenated along a ``circuit_index``
        dimension.  A single-circuit payload without parameter bindings returns the
        legacy flat format (no ``circuit_index`` dimension) for backward compatibility.

        Args:
            payload: JobPayload specifying n_qubits and shots.

        Returns:
            xr.Dataset: Dataset containing simulated states, counts, and frequencies.
        """
        n_qubits = payload.n_qubits

        # Simulate execution latency
        time.sleep(random.uniform(0.1, 0.5))

        sub_datasets: list[xr.Dataset] = []

        for circ in payload.circuits:
            circ_shots = circ.shots if circ.shots is not None else payload.shots
            param_sets = circ.parameter_values or [None]

            for _params in param_sets:
                ds = self._simulate_single(n_qubits, circ_shots, payload.meas_level)
                sub_datasets.append(ds)

        # Backward compatible: single result → flat dataset (no circuit_index)
        if len(sub_datasets) == 1:
            ds = sub_datasets[0]
            ds.attrs.update(
                {
                    "shots": circ_shots,
                    "n_qubits": n_qubits,
                    "backend": self.name,
                    "meas_level": payload.meas_level,
                    "meas_return": payload.meas_return,
                }
            )
            return ds

        # Multiple results → concat along circuit_index
        combined = xr.concat(sub_datasets, dim="circuit_index")
        combined.attrs.update(
            {
                "shots": payload.shots,
                "n_qubits": n_qubits,
                "backend": self.name,
                "meas_level": payload.meas_level,
                "meas_return": payload.meas_return,
            }
        )
        return combined

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _simulate_single(n_qubits: int, shots: int, meas_level: int) -> xr.Dataset:
        """Simulate a single circuit execution and return a flat xr.Dataset."""
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

            if meas_level == 2:
                # Level 2: counts — store as complex with zero imaginary part
                data_vars[str(i)] = xr.DataArray(
                    [complex(val, 0.0) for val in qubit_vals],
                    dims=[f"acq_index_{i}"],
                    coords={f"acq_index_{i}": list(range(shots))},
                )
            else:
                # Level 0/1: IQ data — simulate with small Gaussian noise
                iq_real = np.array(qubit_vals) + np.random.normal(0, 0.05, shots)
                iq_imag = np.random.normal(0, 0.05, shots)
                data_vars[str(i)] = xr.DataArray(
                    [complex(r, im) for r, im in zip(iq_real, iq_imag)],
                    dims=[f"acq_index_{i}"],
                    coords={f"acq_index_{i}": list(range(shots))},
                )

        return xr.Dataset(data_vars)
