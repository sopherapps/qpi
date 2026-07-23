"""Unit tests for the shared event envelope (RFC 0001 §6).

These verify the driver-side envelope serialises to, and parses from, the same
JSON shape QPI-UI puts on the wire, and that defaults (id, timestamp) are filled
in for outbound events.
"""

import json

from qpi_driver.events import Event, EventType


def test_new_event_assigns_defaults():
    event = Event(
        type=EventType.JOB_RESULT, driver="drv_test", payload={"job_id": "job_1"}
    )

    assert event.id.startswith("evt_")
    assert event.ts.endswith("Z")
    assert event.driver == "drv_test"
    assert event.type is EventType.JOB_RESULT


def test_event_serialises_to_wire_shape():
    event = Event(
        type=EventType.JOB_RESULT,
        driver="drv_test",
        payload={"job_id": "job_1", "status": "completed"},
        id="evt_abc",
        ts="2026-07-22T10:04:05.123Z",
    )

    data = json.loads(event.to_json())

    assert data == {
        "id": "evt_abc",
        "driver": "drv_test",
        "type": "JobResult",
        "ts": "2026-07-22T10:04:05.123Z",
        "payload": {"job_id": "job_1", "status": "completed"},
    }


def test_event_round_trips_through_json():
    event = Event(
        type=EventType.JOB_DISPATCH, driver="drv_test", payload={"job_id": "job_1"}
    )

    back = Event.from_json(event.to_json())

    assert back.id == event.id
    assert back.driver == event.driver
    assert back.type is EventType.JOB_DISPATCH
    assert back.ts == event.ts
    assert back.payload == {"job_id": "job_1"}


def test_event_parses_server_envelope():
    """An envelope produced by QPI-UI (string type, bytes) parses cleanly."""
    raw = json.dumps(
        {
            "id": "evt_from_server",
            "driver": "drv_test",
            "type": "JobDispatch",
            "ts": "2026-07-22T10:04:05.123Z",
            "payload": {"job_id": "job_1", "circuits": []},
        }
    ).encode()

    event = Event.from_json(raw)

    assert event.type is EventType.JOB_DISPATCH
    assert event.id == "evt_from_server"
    assert event.payload["job_id"] == "job_1"
