/**
 * Smoke test for the JavaScript QPI client SDK.
 *
 * Usage:
 *   QPI_BASE_URL=http://localhost:8090 QPI_API_TOKEN=xxx node smoke_js.mjs
 */

import { QPIClient } from "../qpi-client/js/dist/index.js";

const baseUrl = process.env.QPI_BASE_URL || "http://127.0.0.1:8090";
const apiToken = process.env.QPI_API_TOKEN || "test-api-token-abc-123";

const client = new QPIClient({ baseUrl, apiToken });

async function main() {
  // List jobs
  const jobs = await client.listJobs();
  console.log(`JS client list_jobs returned ${jobs.length} jobs`);

  // Get first job if any
  if (jobs.length > 0) {
    const job = await client.getJob(jobs[0].id);
    console.log(`JS client get_job returned status=${job.status}`);
  }

  console.log("JS client smoke test passed");
}

main().catch((err) => {
  console.error("JS client smoke test failed:", err);
  process.exit(1);
});
