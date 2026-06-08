from qpi_driver.cli import app
from typer.testing import CliRunner

runner = CliRunner()


def test_cli_version():
    """Verify that the version command outputs 1.0.0."""
    result = runner.invoke(app, ["version"])
    assert result.exit_code == 0
    assert "1.0.0" in result.stdout


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
    assert "Error: registration token is required" in output
