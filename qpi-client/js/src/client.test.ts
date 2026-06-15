/**
 * Unit tests for the QPIClient class.
 *
 * These tests use a lightweight mock of the global `fetch` function so that
 * the suite can run in any Node environment without starting a real server.
 */

import { QPIClient, QPIError } from "./client.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function mockFetch(response: {
  ok?: boolean;
  status?: number;
  statusText?: string;
  json?: () => Promise<unknown>;
  text?: () => Promise<string>;
}): jest.SpyInstance {
  return jest.spyOn(globalThis, "fetch").mockResolvedValue({
    ok: true,
    status: 200,
    statusText: "OK",
    json: async () => ({}),
    text: async () => "",
    ...response,
  } as Response);
}

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------

describe("QPIClient construction", () => {
  it("stores baseUrl without trailing slash", () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090/" });
    // baseUrl is private, but we can verify via the requests it makes.
    const spy = mockFetch({ json: async () => ({ id: "j1" }) });
    return client.getJob("j1").then(() => {
      const url = spy.mock.calls[0][0] as string;
      expect(url).toBe("http://localhost:8090/api/jobs/j1");
      spy.mockRestore();
    });
  });

  it("sends X-API-Token header when apiToken is provided", () => {
    const client = new QPIClient({
      baseUrl: "http://localhost:8090",
      apiToken: "secret",
    });
    const spy = mockFetch({ json: async () => ({ id: "j1" }) });
    return client.getJob("j1").then(() => {
      const init = spy.mock.calls[0][1] as RequestInit;
      expect(init.headers).toMatchObject({
        "Content-Type": "application/json",
        "X-API-Token": "secret",
      });
      spy.mockRestore();
    });
  });

  it("does not send X-API-Token when omitted", () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ id: "j1" }) });
    return client.getJob("j1").then(() => {
      const init = spy.mock.calls[0][1] as RequestInit;
      expect(init.headers).not.toHaveProperty("X-API-Token");
      spy.mockRestore();
    });
  });
});

// ---------------------------------------------------------------------------
// submitJob
// ---------------------------------------------------------------------------

describe("submitJob", () => {
  it("returns the job id from job_id field", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ job_id: "abc-123" }) });

    const id = await client.submitJob({
      circuits: [{ circuit: "OPENQASM 3.0;" }],
    });

    expect(id).toBe("abc-123");
    spy.mockRestore();
  });

  it("returns the job id from id field", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ id: "xyz-789" }) });

    const id = await client.submitJob({
      circuits: [{ circuit: "OPENQASM 3.0;" }],
    });

    expect(id).toBe("xyz-789");
    spy.mockRestore();
  });

  it("throws when response lacks an id", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ other: "data" }) });

    await expect(
      client.submitJob({ circuits: [{ circuit: "qasm" }] }),
    ).rejects.toThrow(/did not contain a job ID/);

    spy.mockRestore();
  });

  it("sends the correct JSON payload", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ job_id: "j1" }) });

    await client.submitJob({
      circuits: [{ circuit: "qasm", shots: 512 }],
      shots: 1024,
      meas_level: 1,
      meas_return: "avg",
      qpu_target: "qpu-01",
    });

    const init = spy.mock.calls[0][1] as RequestInit;
    const body = JSON.parse(init.body as string);
    expect(body.circuits).toEqual([{ circuit: "qasm", shots: 512 }]);
    expect(body.shots).toBe(1024);
    expect(body.meas_level).toBe(1);
    expect(body.meas_return).toBe("avg");
    expect(body.qpu_target).toBe("qpu-01");

    spy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// getJob
// ---------------------------------------------------------------------------

describe("getJob", () => {
  it("returns the parsed job record", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({
      json: async () => ({
        id: "j1",
        status: "completed",
        results: { counts: { "0x0": 100 } },
        duration: 1.23,
      }),
    });

    const job = await client.getJob("j1");

    expect(job.id).toBe("j1");
    expect(job.status).toBe("completed");
    expect(job.duration).toBe(1.23);
    spy.mockRestore();
  });

  it("URL-encodes the job id", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ id: "j1" }) });

    await client.getJob("job/with#special");

    const url = spy.mock.calls[0][0] as string;
    expect(url).toBe("http://localhost:8090/api/jobs/job%2Fwith%23special");
    spy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// listJobs
// ---------------------------------------------------------------------------

describe("listJobs", () => {
  it("handles a bare array response", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({
      json: async () => [
        { id: "j1", status: "pending" },
        { id: "j2", status: "completed" },
      ],
    });

    const jobs = await client.listJobs();

    expect(jobs).toHaveLength(2);
    expect(jobs[0].id).toBe("j1");
    spy.mockRestore();
  });

  it("handles a wrapped { jobs: [...] } response", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({
      json: async () => ({ jobs: [{ id: "j3", status: "running" }] }),
    });

    const jobs = await client.listJobs();

    expect(jobs).toHaveLength(1);
    expect(jobs[0].id).toBe("j3");
    spy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// cancelJob
// ---------------------------------------------------------------------------

describe("cancelJob", () => {
  it("returns the updated job record", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({
      json: async () => ({ id: "j1", status: "cancelled" }),
    });

    const job = await client.cancelJob("j1");

    expect(job.status).toBe("cancelled");
    spy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// waitForJob
// ---------------------------------------------------------------------------

describe("waitForJob", () => {
  it("resolves immediately when job is already terminal", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({
      json: async () => ({ id: "j1", status: "completed" }),
    });

    const job = await client.waitForJob("j1");

    expect(job.status).toBe("completed");
    expect(spy).toHaveBeenCalledTimes(1);
    spy.mockRestore();
  });

  it("polls until the job reaches a terminal state", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({})
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: "OK",
        json: async () => ({ id: "j1", status: "running" }),
        text: async () => "",
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: "OK",
        json: async () => ({ id: "j1", status: "completed" }),
        text: async () => "",
      } as Response);

    const job = await client.waitForJob("j1", { interval: 10 });

    expect(job.status).toBe("completed");
    expect(spy).toHaveBeenCalledTimes(2);
    spy.mockRestore();
  }, 10000);

  it("throws when timeout is exceeded", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({
      json: async () => ({ id: "j1", status: "running" }),
    });

    await expect(
      client.waitForJob("j1", { interval: 10, timeout: 25 }),
    ).rejects.toThrow(/did not complete within/);
    spy.mockRestore();
  }, 10000);
});

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

describe("HTTP error handling", () => {
  it("throws QPIError on non-2xx responses", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({
      ok: false,
      status: 403,
      statusText: "Forbidden",
      text: async () => "insufficient QPU seconds",
    });

    await expect(client.getJob("j1")).rejects.toThrow(QPIError);
    await expect(client.getJob("j1")).rejects.toThrow(/403/);

    spy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// New client methods
// ---------------------------------------------------------------------------

describe("QPU discovery and management", () => {
  it("listQpus calls GET /api/qpus", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => [{ name: "qpu-01" }] });
    const qpus = await client.listQpus();
    expect(qpus).toEqual([{ name: "qpu-01" }]);
    expect(spy.mock.calls[0][0]).toBe("http://localhost:8090/api/qpus");
    spy.mockRestore();
  });

  it("getQpu calls GET /api/qpus/{name}", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ name: "qpu-01" }) });
    const qpu = await client.getQpu("qpu-01");
    expect(qpu).toEqual({ name: "qpu-01" });
    expect(spy.mock.calls[0][0]).toBe("http://localhost:8090/api/qpus/qpu-01");
    spy.mockRestore();
  });

  it("createQpu calls POST /api/op/qpus/create", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ id: "qpu-123", access_token: "qpi_abc" }) });
    const resp = await client.createQpu({
      name: "qpu-02",
      executor_type: "mock",
    });
    expect(resp).toEqual({ id: "qpu-123", access_token: "qpi_abc" });
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/op/qpus/create",
    );
    const init = spy.mock.calls[0][1] as RequestInit;
    expect(JSON.parse(init.body as string)).toEqual({
      name: "qpu-02",
      executor_type: "mock",
    });
    spy.mockRestore();
  });

  it("connectQpu calls POST /api/op/qpus/connect", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ status: "success", nng_command_port: 6000 }) });
    const resp = await client.connectQpu({
      name: "qpu-02",
      access_token: "token123",
      executor_type: "mock",
    });
    expect(resp).toEqual({ status: "success", nng_command_port: 6000 });
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/op/qpus/connect",
    );
    const init = spy.mock.calls[0][1] as RequestInit;
    expect(JSON.parse(init.body as string)).toEqual({
      name: "qpu-02",
      access_token: "token123",
      executor_type: "mock",
    });
    spy.mockRestore();
  });

  it("toggleQpu calls POST /api/op/qpu/toggle", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ success: true }) });
    const resp = await client.toggleQpu("qpu-123", true);
    expect(resp).toEqual({ success: true });
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/op/qpu/toggle",
    );
    const init = spy.mock.calls[0][1] as RequestInit;
    expect(JSON.parse(init.body as string)).toEqual({
      id: "qpu-123",
      enabled: true,
    });
    spy.mockRestore();
  });
});

describe("Notifications", () => {
  it("listNotifications handles array and items wrappers", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ items: [{ id: "n1" }] }) });
    const notes = await client.listNotifications();
    expect(notes).toEqual([{ id: "n1" }]);
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/collections/notifications/records",
    );
    spy.mockRestore();
  });

  it("dismissNotification calls POST /api/notifications/{id}/dismiss", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    const spy = mockFetch({ json: async () => ({ status: "dismissed" }) });
    const resp = await client.dismissNotification("n1");
    expect(resp).toEqual({ status: "dismissed" });
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/notifications/n1/dismiss",
    );
    spy.mockRestore();
  });
});

describe("Booking Slots", () => {
  it("listTimeSlots, createTimeSlot, updateTimeSlot, deleteTimeSlot work", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });

    // list
    let spy = mockFetch({ json: async () => ({ items: [{ id: "s1" }] }) });
    const slots = await client.listTimeSlots();
    expect(slots).toEqual([{ id: "s1" }]);
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/collections/time_slots/records",
    );
    spy.mockRestore();

    // create
    spy = mockFetch({ json: async () => ({ id: "s1" }) });
    const newSlot = await client.createTimeSlot({
      start_time: "2026-06-14T12:00:00Z",
      end_time: "2026-06-14T13:00:00Z",
    });
    expect(newSlot).toEqual({ id: "s1" });
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/collections/time_slots/records",
    );
    spy.mockRestore();

    // update
    spy = mockFetch({ json: async () => ({ id: "s1", end_time: "new" }) });
    const updated = await client.updateTimeSlot("s1", { end_time: "new" });
    expect(updated).toEqual({ id: "s1", end_time: "new" });
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/collections/time_slots/records/s1",
    );
    expect((spy.mock.calls[0][1] as RequestInit).method).toBe("PATCH");
    spy.mockRestore();

    // delete
    spy = mockFetch({ status: 204 });
    await client.deleteTimeSlot("s1");
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/collections/time_slots/records/s1",
    );
    expect((spy.mock.calls[0][1] as RequestInit).method).toBe("DELETE");
    spy.mockRestore();
  });
});

describe("QPU Time Requests", () => {
  it("listTimeRequests, createTimeRequest, updateTimeRequest work", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });

    // list
    let spy = mockFetch({ json: async () => ({ items: [{ id: "tr1" }] }) });
    const reqs = await client.listTimeRequests();
    expect(reqs).toEqual([{ id: "tr1" }]);
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/collections/qpu_time_requests/records",
    );
    spy.mockRestore();

    // create
    spy = mockFetch({ json: async () => ({ id: "tr1" }) });
    const newReq = await client.createTimeRequest({
      seconds: 100,
      requested_reason: "test",
    });
    expect(newReq).toEqual({ id: "tr1" });
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/collections/qpu_time_requests/records",
    );
    spy.mockRestore();

    // update
    spy = mockFetch({ json: async () => ({ id: "tr1", status: "approved" }) });
    const updated = await client.updateTimeRequest("tr1", {
      status: "approved",
    });
    expect(updated).toEqual({ id: "tr1", status: "approved" });
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/collections/qpu_time_requests/records/tr1",
    );
    expect((spy.mock.calls[0][1] as RequestInit).method).toBe("PATCH");
    spy.mockRestore();
  });
});

describe("Admin User Management", () => {
  it("listUsers and allocateQpuTime work", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });

    // list
    let spy = mockFetch({
      json: async () => ({ items: [{ id: "u1", name: "Alice" }] }),
    });
    const users = await client.listUsers();
    expect(users).toEqual([{ id: "u1", name: "Alice" }]);
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/collections/users/records",
    );
    spy.mockRestore();

    // allocate
    spy = mockFetch({ json: async () => ({ id: "u1", qpu_seconds: 1000 }) });
    const updated = await client.allocateQpuTime("u1", 1000);
    expect(updated).toEqual({ id: "u1", qpu_seconds: 1000 });
    expect(spy.mock.calls[0][0]).toBe(
      "http://localhost:8090/api/admin/users/u1",
    );
    expect((spy.mock.calls[0][1] as RequestInit).method).toBe("PATCH");
    spy.mockRestore();
  });
});

describe("Auth helpers", () => {
  it("authWithPassword updates Authorization header", async () => {
    const client = new QPIClient({ baseUrl: "http://localhost:8090" });
    let spy = mockFetch({
      json: async () => ({ token: "token123", record: { id: "u1" } }),
    });
    const resp = await client.authWithPassword("alice@test.com", "pass123");
    expect(resp).toEqual({ token: "token123", record: { id: "u1" } });
    spy.mockRestore();

    // subsequent requests should have Authorization header
    spy = mockFetch({ json: async () => ({ name: "qpu-01" }) });
    await client.getQpu("qpu-01");
    expect((spy.mock.calls[0][1] as RequestInit).headers).toMatchObject({
      Authorization: "Bearer token123",
    });
    spy.mockRestore();
  });
});
