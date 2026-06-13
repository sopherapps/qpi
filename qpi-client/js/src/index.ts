/**
 * @qpi/client — JavaScript/TypeScript SDK for the QPI quantum computing platform.
 *
 * @example
 * ```typescript
 * import { QPIClient } from "@qpi/client";
 *
 * const client = new QPIClient({
 *   baseUrl: "http://localhost:8090",
 *   apiToken: "my-token",
 * });
 *
 * const jobId = await client.submitJob({
 *   circuits: [{ circuit: "OPENQASM 3.0; ..." }],
 *   shots: 1024,
 * });
 *
 * const result = await client.waitForJob(jobId);
 * console.log(result);
 * ```
 *
 * @packageDocumentation
 */
export {
  QPIClient,
  type QPIClientOptions,
  type CircuitPayload,
  type JobSubmitRequest,
  type JobRecord,
} from "./client.js";
