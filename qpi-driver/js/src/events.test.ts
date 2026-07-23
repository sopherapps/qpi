import { Event, EventType } from "./events.js";

describe("Event", () => {
  test("fills in an id and timestamp when not given", () => {
    const event = new Event(EventType.JobResult, { job_id: "j1" });
    expect(event.id).toMatch(/^evt_[0-9a-f]{24}$/);
    expect(event.ts).toMatch(/Z$/);
    expect(event.driver).toBe("");
  });

  test("toWire produces the shared envelope shape", () => {
    const event = new Event(
      EventType.CryostatReading,
      { readings: {} },
      { driver: "cryo", id: "evt_1", ts: "2026-01-01T00:00:00.000Z" },
    );
    expect(event.toWire()).toEqual({
      id: "evt_1",
      driver: "cryo",
      type: "CryostatReading",
      ts: "2026-01-01T00:00:00.000Z",
      payload: { readings: {} },
    });
  });

  test("JSON round-trips through fromJSON", () => {
    const original = new Event(
      EventType.JobDispatch,
      { job_id: "j1" },
      {
        driver: "qpu_1",
      },
    );
    const decoded = Event.fromJSON(original.toJSON());
    expect(decoded.type).toBe(EventType.JobDispatch);
    expect(decoded.driver).toBe("qpu_1");
    expect(decoded.payload).toEqual({ job_id: "j1" });
  });

  test("fromJSON rejects an envelope with no type", () => {
    expect(() => Event.fromJSON(JSON.stringify({ payload: {} }))).toThrow();
  });
});
