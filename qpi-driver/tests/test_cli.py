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
        expected_version = "0.0.40"
    assert expected_version in result.stdout


def test_cli_start_requires_token():
    """Verify that start command fails if token is not supplied."""
    result = runner.invoke(app, ["start", "--ca-fingerprint", "test-fingerprint"])
    assert result.exit_code == 1
    # Check stdout, stderr, or combined output
    output = result.stdout or ""
    try:
        if result.stderr:
            output += "\n" + result.stderr
    except ValueError:
        pass
    assert "Error: access token is required" in output


def test_cli_start_unsafe_data_dir():
    """Verify that start command fails if data_dir is in an unsafe location."""
    result = runner.invoke(
        app,
        [
            "start",
            "--token",
            "test-token",
            "--ca-fingerprint",
            "test-fingerprint",
            "--data-dir",
            "/var",
        ],
    )
    assert result.exit_code in (1, 2)


def test_cli_start_unsafe_ca_file():
    """Verify that start command fails if ca_file is in an unsafe location."""
    result = runner.invoke(
        app,
        [
            "start",
            "--token",
            "test-token",
            "--ca-fingerprint",
            "test-fingerprint",
            "--ca-file",
            "/",
        ],
    )
    assert result.exit_code in (1, 2)


def test_validate_safe_path():
    """Directly test the validate_safe_path logic for safe and unsafe paths."""
    from pathlib import Path

    import typer
    from qpi_driver.cli import _validate_safe_path

    # Safe paths should not raise typer.Exit
    _validate_safe_path(Path("./bin/data"), "test-dir")
    _validate_safe_path(Path("/tmp/safe-dir"), "test-dir")
    _validate_safe_path(Path("/var/tmp/safe-dir"), "test-dir")
    _validate_safe_path(Path("/var/qpi-driver"), "test-dir")
    _validate_safe_path(Path("/var/qpi-driver/sub"), "test-dir")
    _validate_safe_path(Path("/etc/qpi-driver/sub"), "test-dir")
    _validate_safe_path(Path.home() / "qpi-driver", "test-dir")

    # Root path should raise typer.Exit(code=1)
    with pytest.raises(typer.Exit) as exc:
        _validate_safe_path(Path("/"), "test-dir")
    assert exc.value.exit_code == 1

    # Forbidden system path should raise typer.Exit(code=1)
    with pytest.raises(typer.Exit) as exc:
        _validate_safe_path(Path("/usr/local/bin"), "test-dir")
    assert exc.value.exit_code == 1

    # System roots that are blocked
    with pytest.raises(typer.Exit) as exc:
        _validate_safe_path(Path("/var"), "test-dir")
    assert exc.value.exit_code == 1

    with pytest.raises(typer.Exit) as exc:
        _validate_safe_path(Path("/home"), "test-dir")
    assert exc.value.exit_code == 1

    with pytest.raises(typer.Exit) as exc:
        _validate_safe_path(Path("/Users"), "test-dir")
    assert exc.value.exit_code == 1

    # Home root itself is blocked
    with pytest.raises(typer.Exit) as exc:
        _validate_safe_path(Path.home(), "test-dir")
    assert exc.value.exit_code == 1
