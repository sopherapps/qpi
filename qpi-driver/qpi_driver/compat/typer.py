"""Module containing compatibility imports for typer-related libraries"""

from qpi_driver.compat.shared import BasicCompatClass

try:
    from typer import Exit, Option, Typer, echo

    IS_TYPER_INSTALLED: bool = True
except ImportError:
    IS_TYPER_INSTALLED: bool = False

    class Option(BasicCompatClass): ...

    class Typer(BasicCompatClass): ...

    class Exit(BasicCompatClass): ...

    def echo(*args, **kwargs):
        pass
