import importlib.metadata
import importlib.util

import pytest

has_typer = importlib.util.find_spec("typer") is not None

pytestmark = pytest.mark.skipif(
    not has_typer, reason="typer must be installed to run CLI tests"
)

if has_typer:
    from qpi_driver.cli import app
    from typer.testing import CliRunner

    runner = CliRunner()


def test_cli_version():
    """Verify that the version command outputs the correct package version."""
    result = runner.invoke(app, ["version"])
    assert result.exit_code == 0
    try:
        expected_version = importlib.metadata.version("qpi-driver")
    except importlib.metadata.PackageNotFoundError:
        expected_version = "0.0.5"
    assert expected_version in result.stdout


def test_cli_start_requires_token():
    """Verify that start command fails if token is not supplied."""
    result = runner.invoke(app, ["start"])
    assert result.exit_code == 1
    # Check stdout, stderr, or combined output
    output = result.stdout or ""
    try:
        if result.stderr:
            output += "\n" + result.stderr
    except ValueError:
        pass
    assert "Error: access token is required" in output
