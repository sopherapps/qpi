"""Module containing compatibility imports for quantify-related libraries"""

from enum import Enum
from typing import Any, TypeAlias

try:
    from qblox_instruments import Cluster, ClusterType, InstrumentType
    from qcodes.instrument import InstrumentModule
    from qcodes.instrument.base import Instrument
    from qcodes.parameters import ParameterBase
    from quantify_core.data.handling import set_datadir
    from quantify_scheduler import Schedule
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
        X,
        Y,
        Z,
    )
    from quantify_scheduler.qblox import ClusterComponent

    IS_QUANTIFY_INSTALLED: bool = True

except ImportError:
    IS_QUANTIFY_INSTALLED: bool = False

    class _BasicClass:
        def __init__(self, *args, **kwargs):
            pass

    def set_datadir(*args, **kwargs):
        pass

    QbloxHardwareCompilationConfig: TypeAlias = Any
    ClusterComponent: TypeAlias = Any
    InstrumentCoordinator: TypeAlias = Any
    BasicTransmonElement: TypeAlias = Any
    QuantumDevice: TypeAlias = Any
    SerialCompiler: TypeAlias = Any
    Cluster: TypeAlias = Any

    class CNOT(_BasicClass): ...

    class CZ(_BasicClass): ...

    class H(_BasicClass): ...

    class Measure(_BasicClass): ...

    class Reset(_BasicClass): ...

    class Rxy(_BasicClass): ...

    class Rz(_BasicClass): ...

    class X(_BasicClass): ...

    class Y(_BasicClass): ...

    class Z(_BasicClass): ...

    class InstrumentModule(_BasicClass): ...

    class ParameterBase(_BasicClass): ...

    class ClusterDescription(_BasicClass): ...

    class InstrumentType(Enum): ...

    class ClusterType(Enum): ...

    class Schedule(_BasicClass):
        def add(self, *args, **kwargs):
            pass

    class Instrument(_BasicClass):
        @classmethod
        def close_all(cls):
            pass
