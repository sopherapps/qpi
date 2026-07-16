import atexit
from contextlib import suppress

import pytest

# Unregister QCoDeS' default atexit handler that closes instruments.
# During test execution, standard stdout/stderr/logging handlers might be closed
# by the time the interpreter exits, leading to "ValueError: I/O operation on closed file."
# when QCoDeS attempts to log the closing of instruments.
try:
    from qcodes.instrument.base import Instrument

    atexit.unregister(Instrument.close_all)
except ImportError:
    pass


@pytest.fixture(scope="session", autouse=True)
def cleanup_qcodes_instruments():
    yield
    # Safely close all QCoDeS instruments at the end of the test session,
    # while the logging streams are still open and active.
    try:
        from qcodes.instrument.base import Instrument

        with suppress(Exception):
            Instrument.close_all()
    except ImportError:
        pass
