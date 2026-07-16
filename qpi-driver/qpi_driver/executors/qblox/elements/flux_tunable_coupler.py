from qpi_driver.compat.qblox import CompositeSquareEdge


class FluxTunableCoupler(CompositeSquareEdge):
    """An edge for a flux tunable coupler, labeled as {control_qubit}_{target_qubit}"""

    def generate_edge_config(self) -> dict:
        config = super().generate_edge_config()

        # Override the port name to point directly to the coupler
        parent_name = getattr(self, "parent_element_name", None) or getattr(
            self, "_parent_element_name", None
        )
        child_name = getattr(self, "child_element_name", None) or getattr(
            self, "_child_element_name", None
        )
        coupler_port = f"{parent_name}_{child_name}:fl"

        if self.name in config:
            if "CZ" in config[self.name]:
                op_config = config[self.name]["CZ"]
                if hasattr(op_config, "factory_kwargs"):
                    op_config.factory_kwargs["square_port"] = coupler_port

        return config
