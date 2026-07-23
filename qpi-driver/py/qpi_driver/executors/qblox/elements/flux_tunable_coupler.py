from typing import Literal

from pydantic import Field

from qpi_driver.compat.qblox import (
    ClockResource,
    CompositeSquareEdge,
    Numbers,
    Parameter,
    SchedulerSubmodule,
    ShiftClockPhase,
    SquarePulse,
    TimeableSchedule,
)


class EdgeClockFrequencies(SchedulerSubmodule):
    cz: float = Parameter(
        docstring="Clock frequency for the CZ gate square pulse.",
        unit="Hz",
        initial_value=0.0,
        vals=Numbers(min_value=-1e12, max_value=1e12, allow_nan=True),
    )


class FluxTunableCoupler(CompositeSquareEdge):
    """An edge for a flux tunable coupler, labeled as {control_qubit}_{target_qubit}"""

    edge_type: Literal["FluxTunableCoupler"] = "FluxTunableCoupler"
    clock_freqs: EdgeClockFrequencies = Field(
        default_factory=lambda: EdgeClockFrequencies(name="clock_freqs")
    )

    def generate_edge_config(self) -> dict:
        config = super().generate_edge_config()

        if self.name in config:
            if "CZ" in config[self.name]:
                op_config = config[self.name]["CZ"]
                op_config.factory_func = compile_cz
                if hasattr(op_config, "factory_kwargs"):
                    op_config.factory_kwargs["square_port"] = f"{self.name}:fl"
                    op_config.factory_kwargs["square_clock"] = f"{self.name}.cz"
                    op_config.factory_kwargs["square_freq"] = self.clock_freqs.cz

        return config


def compile_cz(
    square_amp: float,
    square_duration: float,
    square_port: str,
    square_clock: str,
    square_freq: float,
    virt_z_parent_qubit_phase: float,
    virt_z_parent_qubit_clock: str,
    virt_z_child_qubit_phase: float,
    virt_z_child_qubit_clock: str,
    t0: float = 0,
):
    sched = TimeableSchedule("CZ")
    sched.add_resource(ClockResource(name=square_clock, freq=square_freq))

    pulse = sched.add(
        SquarePulse(
            amplitude=square_amp,
            duration=square_duration,
            port=square_port,
            clock=square_clock,
            t0=t0,
        )
    )

    sched.add(
        ShiftClockPhase(
            phase_shift=virt_z_parent_qubit_phase,
            clock=virt_z_parent_qubit_clock,
            t0=t0,
        ),
        ref_op=pulse,
        ref_pt="start",
    )
    sched.add(
        ShiftClockPhase(
            phase_shift=virt_z_child_qubit_phase,
            clock=virt_z_child_qubit_clock,
            t0=t0,
        ),
        ref_op=pulse,
        ref_pt="start",
    )

    return sched
