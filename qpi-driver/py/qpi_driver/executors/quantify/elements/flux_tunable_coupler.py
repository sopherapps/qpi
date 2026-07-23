from qpi_driver.compat.quantify import (
    ClockResource,
    CompositeSquareEdge,
    InstrumentChannel,
    ManualParameter,
    Numbers,
    Schedule,
    ShiftClockPhase,
    SquarePulse,
)


class EdgeClockFrequencies(InstrumentChannel):
    def __init__(self, parent, name, **kwargs):
        super().__init__(parent, name)

        self.add_parameter(
            "cz",
            parameter_class=ManualParameter,
            unit="Hz",
            initial_value=kwargs.get("cz", 0.0),
            vals=Numbers(min_value=-1e12, max_value=1e12, allow_nan=True),
        )


class FluxTunableCoupler(CompositeSquareEdge):
    """An edge for a flux tunable coupler, labeled as {control_qubit}_{target_qubit}"""

    def __init__(self, parent_element_name: str, child_element_name: str, **kwargs):
        clock_freqs_data = kwargs.pop("clock_freqs", {})
        super().__init__(parent_element_name, child_element_name, **kwargs)
        self.clock_freqs = EdgeClockFrequencies(self, "clock_freqs", **clock_freqs_data)
        self.add_submodule("clock_freqs", self.clock_freqs)

    def generate_edge_config(self) -> dict:
        config = super().generate_edge_config()

        if self.name in config:
            if "CZ" in config[self.name]:
                op_config = config[self.name]["CZ"]
                op_config.factory_func = compile_cz
                if hasattr(op_config, "factory_kwargs"):
                    op_config.factory_kwargs["square_port"] = f"{self.name}:fl"
                    op_config.factory_kwargs["square_clock"] = f"{self.name}.cz"
                    op_config.factory_kwargs["square_freq"] = self.clock_freqs.cz()

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
    sched = Schedule("CZ")
    sched.add_resource(ClockResource(name=square_clock, freq=square_freq))

    pulse = SquarePulse(
        amp=square_amp,
        duration=square_duration,
        port=square_port,
        clock=square_clock,
        t0=t0,
    )

    pulse.add_pulse(
        ShiftClockPhase(
            phase_shift=virt_z_parent_qubit_phase,
            clock=virt_z_parent_qubit_clock,
            t0=t0,
        )
    )
    pulse.add_pulse(
        ShiftClockPhase(
            phase_shift=virt_z_child_qubit_phase,
            clock=virt_z_child_qubit_clock,
            t0=t0,
        )
    )

    sched.add(pulse)
    return sched
