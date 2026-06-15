"""Module containing compatibility imports for qblox-scheduler-related libraries"""

from enum import Enum
from typing import Any, TypeAlias

from qpi_driver.compat.shared import BasicCompatClass

try:
    from qblox_instruments import Cluster, ClusterType, InstrumentType  # noqa: F401
    from qblox_scheduler import HardwareAgent, Schedule  # noqa: F401
    from qblox_scheduler.device_under_test.quantum_device import (
        QuantumDevice,  # noqa: F401
    )
    from qblox_scheduler.device_under_test.transmon_element import (
        BasicTransmonElement,  # noqa: F401
    )
    from qblox_scheduler.operations import (
        CNOT,
        CZ,
        H,
        Measure,
        Operation,
        Reset,
        Rxy,
        Rz,
        S,
        SDagger,
        T,
        TDagger,
        X,
        Y,
        Z,
    )
    from qblox_scheduler.qblox.hardware_agent import (
        QbloxHardwareCompilationConfig,  # noqa: F401
    )
    from qcodes.instrument import InstrumentModule  # noqa: F401
    from qcodes.instrument.base import Instrument  # noqa: F401
    from qcodes.parameters import ParameterBase  # noqa: F401

    IS_QBLOX_SCHEDULER_INSTALLED: bool = True

except ImportError:
    IS_QBLOX_SCHEDULER_INSTALLED: bool = False

    HardwareAgent: TypeAlias = Any
    QbloxHardwareCompilationConfig: TypeAlias = Any
    QuantumDevice: TypeAlias = Any
    BasicTransmonElement: TypeAlias = Any
    Cluster: TypeAlias = Any

    class CNOT(BasicCompatClass): ...

    class CZ(BasicCompatClass): ...

    class H(BasicCompatClass): ...

    class Measure(BasicCompatClass): ...

    class Reset(BasicCompatClass): ...

    class Rxy(BasicCompatClass): ...

    class Rz(BasicCompatClass): ...

    class X(BasicCompatClass): ...

    class Y(BasicCompatClass): ...

    class Z(BasicCompatClass): ...

    class S(BasicCompatClass): ...

    class T(BasicCompatClass): ...

    class SDagger(BasicCompatClass): ...

    class TDagger(BasicCompatClass): ...

    class InstrumentModule(BasicCompatClass): ...

    class ParameterBase(BasicCompatClass): ...

    class InstrumentType(Enum): ...

    class ClusterType(Enum): ...

    class Operation(BasicCompatClass): ...

    class Schedule(BasicCompatClass):
        def add(self, *args, **kwargs):
            pass

    class Instrument(BasicCompatClass):
        @classmethod
        def close_all(cls):
            pass
