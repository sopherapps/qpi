import typer
from typing_extensions import Annotated

from qpi_driver.driver import run_driver

app = typer.Typer(help="Quantum Processing Interface (QPI) Hardware Driver CLI")


@app.command()
def start(
    host: Annotated[
        str,
        typer.Option(
            "--host",
            "-H",
            envvar="GO_SERVER_HOST",
            help="LAN IP or hostname of the Go PocketBase server",
        ),
    ] = "127.0.0.1",
    port: Annotated[
        int,
        typer.Option(
            "--port", "-P", envvar="GO_SERVER_PORT", help="PocketBase HTTP port"
        ),
    ] = 8090,
    token: Annotated[
        str,
        typer.Option(
            "--token",
            "-t",
            envvar="REGISTRATION_TOKEN",
            help="Token that matches a qpus.registration_token record",
        ),
    ] = "",
    name: Annotated[
        str,
        typer.Option(
            "--name", "-n", envvar="QPU_NAME", help="Human-readable name for this QPU"
        ),
    ] = "QPU-Sim-01",
    executor: Annotated[
        str,
        typer.Option(
            "--executor",
            "-e",
            envvar="DRIVER_BACKEND",
            help="Which executor backend to use (mock, qiskit_aer, quantify, qblox, presto)",
        ),
    ] = "mock",
    data_dir: Annotated[
        str,
        typer.Option(
            "--data-dir",
            "-d",
            envvar="QPI_DATA_DIR",
            help="Directory where intermediate netCDF datasets are written",
        ),
    ] = "bin/data",
):
    """
    Start the QPI driver.
    """
    if not token:
        typer.echo(
            "Error: registration token is required. "
            "Set it via --token / -t or the REGISTRATION_TOKEN environment variable.",
            err=True,
        )
        raise typer.Exit(code=1)

    run_driver(
        host=host,
        port=port,
        token=token,
        name=name,
        executor=executor,
        data_dir=data_dir,
    )


@app.command()
def version():
    """
    Show the version of the QPI driver.
    """
    typer.echo("1.0.0")


if __name__ == "__main__":
    app()
