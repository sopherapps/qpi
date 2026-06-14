import xarray as xr
from qiskit import QuantumCircuit, qasm2, qasm3


def load_qasm(value: str, num_qubits: int | None = None) -> QuantumCircuit:
    """Loads QASM from the QASM string

    Args:
        value: the QASM string from which to load the circuit
        num_qubits: provides number of physical/virtual qubits

    Returns:
        the QASM circuit as got from the string value passed

    Raises:
        ValueError: Failed to parse QASM circuit: exc
    """
    try:
        try:
            return qasm3.loads(value, num_qubits=num_qubits)
        except Exception:
            return qasm2.loads(value, strict=True)
    except Exception as exc:
        raise ValueError(f"Failed to parse QASM circuit: {exc}") from exc


def memory_to_dataset(
    memory: list[str], n_qubits: int, shots: int, meas_level: int
) -> xr.Dataset:
    """Convert shot memory (list of bitstrings) into an xr.Dataset.

    The returned dataset format depends on ``meas_level``:

    * ``meas_level=2`` – Store as ``complex(bit_value, 0.0)``.
    * ``meas_level=1`` – Synthesize IQ-like complex values with Gaussian noise
      so that downstream IQ processing pipelines can be tested against simulators.
    * ``meas_level=0`` – Not supported for simulators (raises ``NotImplementedError``).

    Args:
        memory: List of bitstring outcomes (e.g. ['00', '01', '11']).
        n_qubits: Number of qubits in the circuit.
        shots: Number of shots.
        meas_level: Measurement level (0=raw IQ, 1=kerneled IQ, 2=counts).

    Returns:
        xr.Dataset with one data variable per qubit.

    Raises:
        NotImplementedError: If ``meas_level=0`` is requested.
    """
    if meas_level == 0:
        raise NotImplementedError(
            "Raw waveform data (meas_level=0) is not available from simulators"
        )

    import numpy as np

    data_vars = {}
    for i in range(n_qubits):
        # Qubit state 0 or 1 for each shot. Qubit 0 is LSB, so index is n_qubits - 1 - i.
        qubit_vals = [float(outcome[n_qubits - 1 - i]) for outcome in memory]

        if meas_level == 1:
            # Synthesize IQ-like complex values with Gaussian noise
            # |0⟩ cluster centred at (0.1, 0.05), |1⟩ cluster centred at (0.9, 0.85)
            iq_vals = []
            for val in qubit_vals:
                noise_i = float(np.random.normal(0, 0.05))
                noise_q = float(np.random.normal(0, 0.05))
                if val < 0.5:  # state |0⟩
                    iq_vals.append(complex(0.1 + noise_i, 0.05 + noise_q))
                else:  # state |1⟩
                    iq_vals.append(complex(0.9 + noise_i, 0.85 + noise_q))
            data_vars[str(i)] = xr.DataArray(
                iq_vals,
                dims=[f"acq_index_{i}"],
                coords={f"acq_index_{i}": list(range(shots))},
            )
        else:
            # meas_level=2: classified counts as complex(bit, 0)
            data_vars[str(i)] = xr.DataArray(
                [complex(val, 0.0) for val in qubit_vals],
                dims=[f"acq_index_{i}"],
                coords={f"acq_index_{i}": list(range(shots))},
            )
    return xr.Dataset(data_vars)
