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
  duration?: number;
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

  // -- QPU discovery & management --------------------------------------------

  /** List all online QPUs. */
  async listQpus(): Promise<any[]> {
    return this.get<any[]>("/api/qpus");
  }

  /** Retrieve a single QPU by name. */
  async getQpu(name: string): Promise<any> {
    return this.get<any>(`/api/qpus/${encodeURIComponent(name)}`);
  }

  /** Create a new QPU record (admin-only). Returns the generated access token. */
  async createQpu(request: {
    name: string;
    executor_type?: string;
    num_qubits?: number;
    enabled?: boolean;
  }): Promise<any> {
    return this.post<any>("/api/op/qpus/create", request);
  }

  /** Connect a QPU driver node using its access token. */
  async connectQpu(request: {
    name: string;
    access_token: string;
    executor_type?: string;
    device_config?: Record<string, any>;
  }): Promise<any> {
    return this.post<any>("/api/op/qpus/connect", request);
  }

  /** Toggle QPU driver state (admin-only). */
  async toggleQpu(id: string, enabled: boolean): Promise<any> {
    return this.post<any>("/api/op/qpu/toggle", { id, enabled });
  }

  // -- Notifications ---------------------------------------------------------

  /** List notifications visible to the authenticated user. */
  async listNotifications(): Promise<any[]> {
    const data = await this.get<any | any[]>(
      "/api/collections/notifications/records",
    );
    return Array.isArray(data) ? data : data.items || [];
  }

  /** Dismiss a notification for the authenticated user. */
  async dismissNotification(id: string): Promise<any> {
    return this.post<any>(
      `/api/notifications/${encodeURIComponent(id)}/dismiss`,
    );
  }

  // -- Booking Slots (time_slots) --------------------------------------------

  /** List all booking slots. */
  async listTimeSlots(): Promise<any[]> {
    const data = await this.get<any>("/api/collections/time_slots/records");
    return data.items || [];
  }

  /** Create a new booking slot. */
  async createTimeSlot(slot: {
    start_time: string;
    end_time: string;
    booked_by?: string;
  }): Promise<any> {
    return this.post<any>("/api/collections/time_slots/records", slot);
  }

  /** Update an existing booking slot. */
  async updateTimeSlot(
    id: string,
    slot: {
      start_time?: string;
      end_time?: string;
    },
  ): Promise<any> {
    return this.patch<any>(
      `/api/collections/time_slots/records/${encodeURIComponent(id)}`,
      slot,
    );
  }

  /** Delete a booking slot. */
  async deleteTimeSlot(id: string): Promise<void> {
    await this.delete<void>(
      `/api/collections/time_slots/records/${encodeURIComponent(id)}`,
    );
  }

  // -- QPU Time Requests -----------------------------------------------------

  /** List QPU time requests. */
  async listTimeRequests(): Promise<any[]> {
    const data = await this.get<any>(
      "/api/collections/qpu_time_requests/records",
    );
    return data.items || [];
  }

  /** Create a new QPU time request. */
  async createTimeRequest(request: {
    seconds: number;
    requested_reason?: string;
  }): Promise<any> {
    return this.post<any>(
      "/api/collections/qpu_time_requests/records",
      request,
    );
  }

  /** Update/Handle a QPU time request (admin-only). */
  async updateTimeRequest(
    id: string,
    request: {
      status: "approved" | "rejected";
      rejection_reason?: string;
    },
  ): Promise<any> {
    return this.patch<any>(
      `/api/collections/qpu_time_requests/records/${encodeURIComponent(id)}`,
      request,
    );
  }

  // -- Admin User Management -------------------------------------------------

  /** List all registered users (admin-only). */
  async listUsers(): Promise<any[]> {
    const data = await this.get<any>("/api/collections/users/records");
    return data.items || [];
  }

  /** Allocate QPU time to a user (admin-only). */
  async allocateQpuTime(userId: string, seconds: number): Promise<any> {
    return this.patch<any>(
      `/api/collections/users/records/${encodeURIComponent(userId)}`,
      {
        qpu_seconds: seconds,
      },
    );
  }

  // -- Auth helpers -----------------------------------------------------------

  /** Authenticate as a regular user using email/password. */
  async authWithPassword(identity: string, password: string): Promise<any> {
    const resp = await this.post<any>(
      "/api/collections/users/auth-with-password",
      {
        identity,
        password,
      },
    );
    if (resp.token) {
      this.headers["Authorization"] = `Bearer ${resp.token}`;
    }
    return resp;
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

  private async patch<T>(path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "PATCH",
      headers: this.headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
    await this.assertOk(res);
    return (await res.json()) as T;
  }

  private async delete<T>(path: string): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "DELETE",
      headers: this.headers,
    });
    await this.assertOk(res);
    if (res.status === 204) {
      return {} as T;
    }
    return (await res.json().catch(() => ({}))) as T;
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
