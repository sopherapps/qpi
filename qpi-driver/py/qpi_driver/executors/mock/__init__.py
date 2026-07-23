from typing import Any

import xarray as xr
from qiskit import transpile
from qiskit.providers.basic_provider import BasicSimulator

from qpi_driver.executors.utils.batch import (
    combine_circuit_datasets,
    iter_circuit_datasets,
)
from qpi_driver.executors.utils.counts import simulator_dataset_to_result
from qpi_driver.executors.utils.qiskit import load_qasm, memory_to_dataset
from qpi_driver.executors.utils.result import build_qiskit_result

from ..base import Executor, JobPayload
from ..utils.types import cast_to


class MockExecutor(Executor):
    def __init__(self, name: str = "mock", **kwargs: Any):
        super().__init__(name, **kwargs)
        self._simulator = BasicSimulator()

    def execute(self, payload: JobPayload) -> xr.Dataset:
        """Execute quantum circuit simulation using Qiskit BasicSimulator.

        Every circuit is run honouring its per-circuit ``shots`` override.  A
        single-circuit payload returns that circuit's flat dataset; multi-circuit
        payloads are bundled so circuits with different classical-bit widths or
        shot counts stay independent (see ``combine_circuit_datasets``).

        Args:
            payload: JobPayload specifying shots and circuits.

        Returns:
            xr.Dataset: Dataset containing measured state outcomes.
        """
        sub_datasets: list[xr.Dataset] = []

        for circ in payload.circuits:
            circ_shots = circ.shots if circ.shots is not None else payload.shots
            qasm_str = circ.circuit
            circuit = load_qasm(qasm_str)

            param_sets = circ.parameter_values or [None]
            for param_vals in param_sets:
                bound_circuit = circuit
                if param_vals is not None and circuit.parameters:
                    bound_circuit = circuit.assign_parameters(param_vals)

                t_qc = transpile(bound_circuit, self._simulator)
                result = self._simulator.run(
                    t_qc, shots=circ_shots, memory=True
                ).result()
                memory = result.get_memory(t_qc)

                ds = memory_to_dataset(
                    memory, circuit.num_clbits, circ_shots, payload.meas_level
                )
                ds.attrs.update(
                    {
                        "shots": circ_shots,
                        "n_qubits": circuit.num_qubits,
                        "backend": self.name,
                        "meas_level": payload.meas_level,
                        "meas_return": payload.meas_return,
                    }
                )
                sub_datasets.append(ds)

        return combine_circuit_datasets(sub_datasets)

    def process_result(self, dataset: xr.Dataset, job_id: str) -> dict:
        """Convert the simulator's xr.Dataset into a Qiskit-compatible result dict.

        Supports meas_level 1 (IQ memory) and 2 (classified counts).
        meas_level 0 is not supported for simulators.

        Args:
            dataset: xr.Dataset from execute().
            job_id: Unique job ID.

        Returns:
            dict: Qiskit-compatible result dict.
        """
        meas_level = cast_to(int, dataset.attrs.get("meas_level"), 2)
        meas_return = str(dataset.attrs.get("meas_return", "single"))
        backend = dataset.attrs.get("backend", self.name)

        circuit_results = [
            simulator_dataset_to_result(sub_ds, meas_level, meas_return)
            for sub_ds in iter_circuit_datasets(dataset)
        ]
        return build_qiskit_result(circuit_results, job_id, backend)
