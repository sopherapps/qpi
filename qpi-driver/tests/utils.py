import json
import os
from typing import Any


def load_json_fixture(fixture_relative_path: str) -> Any:
    """Load a JSON fixture from the tests/fixtures directory."""
    base_dir = os.path.dirname(os.path.abspath(__file__))
    full_path = os.path.join(base_dir, "fixtures", fixture_relative_path)
    with open(full_path, "r") as f:
        return json.load(f)
