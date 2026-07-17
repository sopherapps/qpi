from qpi_driver.compat.qblox import CompositeSquareEdge


class FluxTunableCoupler(CompositeSquareEdge):
    """An edge for a flux tunable coupler, labeled as {control_qubit}_{target_qubit}"""

    def generate_edge_config(self) -> dict:
        config = super().generate_edge_config()

        if self.name in config:
            if "CZ" in config[self.name]:
                op_config = config[self.name]["CZ"]
                if hasattr(op_config, "factory_kwargs"):
                    op_config.factory_kwargs["square_port"] = f"{self.name}:fl"
                    op_config.factory_kwargs["square_clock"] = f"{self.name}.cz"

        return config
