<pre align="center">
 ██████╗ ██████╗ ██╗
██╔═══██╗██╔══██╗██║
██║   ██║██████╔╝██║
██║▄▄ ██║██╔═══╝ ██║
╚██████╔╝██║     ██║
 ╚══▀▀═╝ ╚═╝     ╚═╝
</pre>

<h1 align="center">QPI JavaScript/TypeScript Client</h1>

<p align="center">
  <a href="https://www.npmjs.com/package/qpi-client"><img src="https://badge.fury.io/js/qpi-client.svg" alt="npm version"></a>
  <a href="https://github.com/sopherapps/qpi/actions/workflows/ci.yml"><img src="https://github.com/sopherapps/qpi/actions/workflows/ci.yml/badge.svg" alt="CI/CD Workflow"></a>
  <a href="https://github.com/sopherapps/qpi/releases"><img src="https://img.shields.io/github/v/tag/sopherapps/qpi?label=version" alt="GitHub Tag"></a>
  <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
</p>

<p align="center">
  JavaScript/TypeScript client SDK for the <a href="https://github.com/sopherapps/qpi">QPI</a> quantum computing platform.
  Zero runtime dependencies — uses the global <code>fetch</code> API.
</p>

<p align="center">
  <strong><a href="https://sopherapps.github.io/qpi/clients/javascript/">📚 Read the Documentation</a></strong>
</p>

---

## Install

```bash
npm install qpi-client
```

Requires **Node.js ≥ 18** (or any environment with a global <code>fetch</code>).

---

## Quick Start

```typescript
import { QPIClient } from "qpi-client";

const client = new QPIClient({
  baseUrl: "http://localhost:8090",
  apiToken: "my-api-token",
});

// Submit a job
const jobId = await client.submitJob({
  circuits: [
    {
      circuit: `OPENQASM 3.0;
include "stdgates.inc";
qubit[2] q;
bit[2] c;
h q[0];
cx q[0], q[1];
c = measure q;`,
    },
  ],
  shots: 1024,
});
console.log("Job ID:", jobId);

// Wait for completion
const result = await client.waitForJob(jobId);
console.log("Status:", result.status);
console.log("Results:", result.results);
```

---

## API Overview

| Method | Description |
|--------|-------------|
| `new QPIClient({ baseUrl, apiToken })` | Create a new client |
| `submitJob(request)` | Submit a quantum job |
| `getJob(jobId)` | Retrieve a job by ID |
| `listJobs()` | List all jobs for the authenticated user |
| `cancelJob(jobId)` | Request job cancellation |
| `waitForJob(jobId, options?)` | Poll until the job reaches a terminal state |
| `listQpus()` | List all online QPUs |
| `getQpu(name)` | Retrieve a single QPU |
| `createQpu(request)` | Create a new QPU record (admin) |
| `connectQpu(request)` | Connect a QPU driver node |
| `listTimeSlots()` | List booking slots |
| `listNotifications()` | List visible notifications |

---

## TypeScript

Full type definitions are included. All request/response types are exported:

```typescript
import type {
  QPIClientOptions,
  CircuitPayload,
  JobSubmitRequest,
  JobRecord,
} from "qpi-client";
```

---

## Documentation

- [Main QPI Repository](https://github.com/sopherapps/qpi)
- [npm Package Page](https://www.npmjs.com/package/qpi-client)

---

## License

MIT — see the [main repository](https://github.com/sopherapps/qpi/blob/main/LICENSE) for details.
