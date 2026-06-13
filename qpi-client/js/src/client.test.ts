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
      }),
    });

    const job = await client.getJob("j1");

    expect(job.id).toBe("j1");
    expect(job.status).toBe("completed");
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
