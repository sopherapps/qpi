/**
 * Core QPI client implementation.
 *
 * Uses the global `fetch` API available in Node ≥ 18 and all modern browsers,
 * so there are **zero runtime dependencies**.
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** A single circuit within a job submission. */
export interface CircuitPayload {
  /** OpenQASM 3 string for the circuit. */
  circuit: string;
  /** Optional parameter bind sets (one inner array per bind set). */
  parameter_values?: number[][];
  /** Per-circuit shot override. */
  shots?: number;
}

/** Request body for `POST /api/jobs`. */
export interface JobSubmitRequest {
  circuits: CircuitPayload[];
  /** Default number of shots for every circuit. */
  shots?: number;
  /** Measurement level (2 = classified bits). */
  meas_level?: number;
  /** `"single"` or `"avg"`. */
  meas_return?: string;
  /** Optional QPU routing hint. */
  qpu_target?: string;
}

/** A job record returned by the server. */
export interface JobRecord {
  id: string;
  status:
    | "pending"
    | "queued"
    | "running"
    | "completed"
    | "failed"
    | "cancelled";
  payload: unknown;
  results: unknown;
  created: string;
  updated: string;
}

/** Options for constructing a {@link QPIClient}. */
export interface QPIClientOptions {
  /** Root URL of the QPI orchestrator, e.g. `"http://localhost:8090"`. */
  baseUrl: string;
  /**
   * API token for authentication.  Sent as `X-API-Token` header.
   * Omit when relying on cookie/JWT auth.
   */
  apiToken?: string;
}

/** Options for {@link QPIClient.waitForJob}. */
export interface WaitOptions {
  /** Maximum time (ms) to wait before rejecting. Default: no limit. */
  timeout?: number;
  /** Polling interval in ms. Default: `5000`. */
  interval?: number;
}

// ---------------------------------------------------------------------------
// Error
// ---------------------------------------------------------------------------

/** Error thrown when the QPI API returns a non-2xx response. */
export class QPIError extends Error {
  constructor(
    public readonly status: number,
    public readonly statusText: string,
    public readonly body: string,
  ) {
    super(`QPI API error ${status} ${statusText}: ${body}`);
    this.name = "QPIError";
  }
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/**
 * HTTP client for the QPI orchestrator REST API.
 *
 * All methods are `async` and return parsed JSON responses.  Network and
 * server errors are surfaced as {@link QPIError} instances.
 */
export class QPIClient {
  private readonly baseUrl: string;
  private readonly headers: Record<string, string>;

  constructor(options: QPIClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, "");
    this.headers = { "Content-Type": "application/json" };
    if (options.apiToken) {
      this.headers["X-API-Token"] = options.apiToken;
    }
  }

  // -- public API -----------------------------------------------------------

  /**
   * Submit a new job.
   *
   * @returns The server-assigned job ID.
   */
  async submitJob(request: JobSubmitRequest): Promise<string> {
    const data = await this.post<{ id?: string; job_id?: string }>(
      "/api/jobs",
      request,
    );
    const id = data.id ?? data.job_id;
    if (!id) {
      throw new Error(
        `Server response did not contain a job ID: ${JSON.stringify(data)}`,
      );
    }
    return id;
  }

  /** Retrieve full details for a job. */
  async getJob(jobId: string): Promise<JobRecord> {
    return this.get<JobRecord>(`/api/jobs/${encodeURIComponent(jobId)}`);
  }

  /** List all jobs belonging to the authenticated user. */
  async listJobs(): Promise<JobRecord[]> {
    const data = await this.get<JobRecord[] | { jobs: JobRecord[] }>(
      "/api/jobs",
    );
    return Array.isArray(data) ? data : data.jobs;
  }

  /** Request cancellation of a job. */
  async cancelJob(jobId: string): Promise<JobRecord> {
    return this.post<JobRecord>(
      `/api/jobs/${encodeURIComponent(jobId)}/cancel`,
    );
  }

  /**
   * Poll until a job reaches a terminal state.
   *
   * @returns The final {@link JobRecord}.
   * @throws {Error} If the timeout is exceeded.
   */
  async waitForJob(jobId: string, options?: WaitOptions): Promise<JobRecord> {
    const interval = options?.interval ?? 5_000;
    const deadline = options?.timeout
      ? Date.now() + options.timeout
      : undefined;

    // eslint-disable-next-line no-constant-condition
    while (true) {
      const job = await this.getJob(jobId);
      if (["completed", "failed", "cancelled"].includes(job.status)) {
        return job;
      }
      if (deadline && Date.now() >= deadline) {
        throw new Error(
          `Job ${jobId} did not complete within ${options!.timeout}ms`,
        );
      }
      await sleep(interval);
    }
  }

  // -- internal helpers -----------------------------------------------------

  private async get<T>(path: string): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "GET",
      headers: this.headers,
    });
    await this.assertOk(res);
    return (await res.json()) as T;
  }

  private async post<T>(path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      headers: this.headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
    await this.assertOk(res);
    return (await res.json()) as T;
  }

  private async assertOk(res: Response): Promise<void> {
    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new QPIError(res.status, res.statusText, text);
    }
  }
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
