"""Module containing compatibility imports for quantify-related libraries"""

from enum import Enum
from typing import Any, TypeAlias

from qpi_driver.compat.shared import BasicCompatClass

try:
    from qblox_instruments import Cluster, ClusterType, InstrumentType
    from qcodes.instrument import InstrumentModule
    from qcodes.instrument.base import Instrument
    from qcodes.parameters import ParameterBase
    from quantify_core.data.handling import set_datadir
    from quantify_scheduler import Operation, Schedule
    from quantify_scheduler.backends.graph_compilation import SerialCompiler
    from quantify_scheduler.backends.qblox_backend import (
        QbloxHardwareCompilationConfig,
    )
    from quantify_scheduler.backends.types.qblox import ClusterDescription
    from quantify_scheduler.device_under_test.quantum_device import (
        QuantumDevice,
    )
    from quantify_scheduler.device_under_test.transmon_element import (
        BasicTransmonElement,
    )
    from quantify_scheduler.instrument_coordinator import InstrumentCoordinator
    from quantify_scheduler.operations.gate_library import (
        CNOT,
        CZ,
        H,
        Measure,
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
    from quantify_scheduler.qblox import ClusterComponent

    IS_QUANTIFY_INSTALLED: bool = True

except ImportError:
    IS_QUANTIFY_INSTALLED: bool = False

    def set_datadir(*args, **kwargs):
        pass

    QbloxHardwareCompilationConfig: TypeAlias = Any
    ClusterComponent: TypeAlias = Any
    InstrumentCoordinator: TypeAlias = Any
    BasicTransmonElement: TypeAlias = Any
    QuantumDevice: TypeAlias = Any
    SerialCompiler: TypeAlias = Any
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

    class ClusterDescription(BasicCompatClass): ...

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
