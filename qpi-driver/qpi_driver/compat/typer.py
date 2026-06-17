"""Module containing compatibility imports for typer-related libraries"""

from qpi_driver.compat.shared import BasicCompatClass

try:
    from typer import Exit, Option, Typer, echo

    from rich import print as rich_print
    from rich.panel import Panel

    IS_TYPER_INSTALLED: bool = True
except ImportError:
    IS_TYPER_INSTALLED: bool = False

    class Panel(BasicCompatClass): ...
    class Option(BasicCompatClass): ...

    class Typer(BasicCompatClass): ...

    class Exit(BasicCompatClass): ...

    def echo(*args, **kwargs):
        pass

    def rich_print(*args, **kwargs):
        print(*args, **kwargs)
