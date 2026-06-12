"""Module containing compatibility imports for typer-related libraries"""

try:
    from typer import Exit, Option, Typer, echo

    IS_TYPER_INSTALLED: bool = True
except ImportError:
    IS_TYPER_INSTALLED: bool = False

    def Option(*args, **kwargs):
        pass

    def Typer(*args, **kwargs):
        pass

    def Exit(*args, **kwargs):
        pass

    def echo(*args, **kwargs):
        pass
