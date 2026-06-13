"""Module containing compatibility helpers for Qiskit aer."""

from qpi_driver.compat.shared import BasicCompatClass

try:
    from qiskit_aer import AerSimulator

    IS_AER_INSTALLED: bool = True
except ImportError:
    IS_AER_INSTALLED: bool = False

    class AerSimulator(BasicCompatClass): ...
