"""Utilities for handling types"""

from contextlib import suppress
from typing import Any, Type, TypeVar

T = TypeVar("T")


def cast_to(_type: Type[T], value: Any, default: T) -> T:
    """Cast value to type and return it or returns default if the cast fails.

    Args:
        _type: Type to cast the value to.
        value: Value to cast.
        default: Value to return if cast fails.

    Returns:
        the cast value.
    """
    result = default
    with suppress(ValueError, TypeError):
        result = _type(value)
    return result
