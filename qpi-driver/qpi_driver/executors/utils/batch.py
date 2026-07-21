"""Bundle per-circuit datasets from a batch job into a single dataset.

A batch job runs several circuits, each producing its own ``xr.Dataset``.
Concatenating those along a shared ``circuit_index`` dimension only works when
every circuit has the same data variables and axis sizes; circuits with
different classical-bit or qubit widths (or different shot counts) either raise
or get silently misaligned onto a padded shared axis.

To keep heterogeneous batches independent, each circuit's variables, dimensions
and coordinates are stored under a ``circuit{i}/`` namespace inside one merged
dataset, and each circuit's ``attrs`` (shots, n_qubits, num_clbits, ...) are
kept verbatim. ``iter_circuit_datasets`` reverses this, yielding each circuit's
original dataset back.
"""

from typing import Iterator

import xarray as xr

_NAMESPACE = "circuit{index}/"
_N_CIRCUITS_ATTR = "n_circuits"
_CIRCUIT_ATTRS_ATTR = "circuit_attrs"


def combine_circuit_datasets(sub_datasets: list[xr.Dataset]) -> xr.Dataset:
    """Bundle per-circuit datasets into one, keeping each circuit independent.

    A single-circuit batch is returned unchanged for backward compatibility.
    For multiple circuits, every circuit's variables/dims/coords are moved into
    a ``circuit{i}/`` namespace so circuits with different widths coexist
    without being forced onto a shared axis. Per-circuit ``attrs`` are preserved
    and can be recovered with :func:`iter_circuit_datasets`.

    Args:
        sub_datasets: One dataset per (parameter-bound) circuit in the batch.

    Returns:
        The single dataset unchanged, or a merged dataset carrying all circuits.
    """
    if len(sub_datasets) == 1:
        return sub_datasets[0]

    namespaced: list[xr.Dataset] = []
    circuit_attrs: list[dict] = []
    for index, dataset in enumerate(sub_datasets):
        prefix = _NAMESPACE.format(index=index)
        rename_map = {
            name: f"{prefix}{name}" for name in {*dataset.variables, *dataset.dims}
        }
        renamed = dataset.rename(rename_map)
        renamed.attrs = {}
        namespaced.append(renamed)
        circuit_attrs.append(dict(dataset.attrs))

    combined = xr.merge(namespaced, combine_attrs="drop")
    combined.attrs = dict(sub_datasets[0].attrs)
    combined.attrs[_N_CIRCUITS_ATTR] = len(sub_datasets)
    combined.attrs[_CIRCUIT_ATTRS_ATTR] = circuit_attrs
    return combined


def iter_circuit_datasets(dataset: xr.Dataset) -> Iterator[xr.Dataset]:
    """Yield each circuit's dataset from a dataset built by ``execute``.

    Datasets that were never bundled (single-circuit jobs) are yielded as-is.
    Bundled datasets are split back into their per-circuit datasets, with each
    circuit's namespace stripped and its original ``attrs`` restored on top of
    the shared batch-level ``attrs``.

    Args:
        dataset: A dataset returned by an executor's ``execute``.

    Yields:
        One dataset per circuit, in circuit order.
    """
    n_circuits = int(dataset.attrs.get(_N_CIRCUITS_ATTR, 0))
    if n_circuits <= 0:
        yield dataset
        return

    circuit_attrs = dataset.attrs.get(_CIRCUIT_ATTRS_ATTR) or [{}] * n_circuits
    batch_attrs = {
        key: value
        for key, value in dataset.attrs.items()
        if key not in (_N_CIRCUITS_ATTR, _CIRCUIT_ATTRS_ATTR)
    }

    for index in range(n_circuits):
        prefix = _NAMESPACE.format(index=index)
        circuit_vars = [var for var in dataset.data_vars if str(var).startswith(prefix)]
        circuit = dataset[circuit_vars]
        rename_map = {
            name: str(name)[len(prefix) :]
            for name in {*circuit.variables, *circuit.dims}
            if str(name).startswith(prefix)
        }
        circuit = circuit.rename(rename_map)
        circuit.attrs = {**batch_attrs, **circuit_attrs[index]}
        yield circuit
