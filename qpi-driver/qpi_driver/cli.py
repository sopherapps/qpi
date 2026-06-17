import importlib.metadata
from pathlib import Path
from typing import Annotated

from qpi_driver.compat import typer
from qpi_driver.driver import run_driver

app = None
if typer.IS_TYPER_INSTALLED:
    app = typer.Typer(help="Quantum Processing Interface (QPI) Hardware Driver CLI")

    @app.command()
    def start(
        qpi_addr: Annotated[
            str,
            typer.Option(
                "--qpi-addr",
                "-a",
                envvar="QPI_ADDR",
                help="Full URL of the QPI orchestrator (e.g. http://localhost:8090 or https://qpi.example.com)",
            ),
        ] = "http://127.0.0.1:8090",
        token: Annotated[
            str,
            typer.Option(
                "--token",
                "-t",
                envvar="QPI_ACCESS_TOKEN",
                help="QPU access token matching a qpus.access_token record",
            ),
        ] = "",
        name: Annotated[
            str,
            typer.Option(
                "--name",
                "-n",
                envvar="QPU_NAME",
                help="Human-readable name for this QPU",
            ),
        ] = "qpu_sim_01",
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
            Path,
            typer.Option(
                "--data-dir",
                "-d",
                envvar="QPI_DATA_DIR",
                help="Directory where intermediate pickled datasets are written",
                writable=True,
                readable=True,
                dir_okay=True,
                file_okay=False,
                resolve_path=True,
            ),
        ] = Path("./bin/data"),
        is_dummy: Annotated[
            bool,
            typer.Option(
                "--is-dummy",
                help="Whether to run the executor in dummy/simulation mode",
            ),
        ] = False,
        quantify_hardware_config: Annotated[
            Path,
            typer.Option(
                envvar="QPI_QUANTIFY_HARDWARE_CONFIG",
                help="Path to the quantify hardware-layer configuration file containing specifications about the RF control instruments",
                readable=True,
                dir_okay=False,
                file_okay=True,
                resolve_path=True,
            ),
        ] = Path("./quantify.hardware.json"),
        quantify_device_config: Annotated[
            Path,
            typer.Option(
                envvar="QPI_QUANTIFY_DEVICE_CONFIG",
                help="Path to the quantify device-layer configuration file containing specifications about the chip",
                readable=True,
                dir_okay=False,
                file_okay=True,
                resolve_path=True,
            ),
        ] = Path("./quantify.device.yml"),
        job_timeout: Annotated[
            int,
            typer.Option(
                envvar="QPI_JOB_TIMEOUT",
                help="The number of seconds to wait for a job to complete.",
            ),
        ] = 10,
    ):
        """
        Start the QPI driver.
        """
        if not token:
            typer.echo(
                "Error: access token is required. "
                "Set it via --token / -t or the QPI_ACCESS_TOKEN environment variable.",
                err=True,
            )
            raise typer.Exit(code=1)

        run_driver(
            qpi_addr=qpi_addr,
            token=token,
            name=name,
            executor=executor,
            data_dir=data_dir,
            is_dummy=is_dummy,
            quantify_hardware_config=quantify_hardware_config,
            quantify_device_config=quantify_device_config,
            job_timeout=job_timeout,
        )

    @app.command()
    def version():
        """
        Show the version of the QPI driver.
        """

        try:
            ver = importlib.metadata.version("qpi-driver")
        except importlib.metadata.PackageNotFoundError:
            ver = "0.0.5"

        typer.echo(ver)

    if __name__ == "__main__":
        app()
