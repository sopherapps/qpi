"""
verify.py — End-to-end verification of the QPi control stack.

Steps:
  1. Poll quantum_jobs until all seeded jobs are "completed".
  2. Verify QPU seconds were deducted from the test user.
  3. Test API token authentication for job listing/retrieval.
  4. Test the admin-only user update endpoint.
  5. Test job cancellation via the API.
  6. Test the recovery engine: manually mark one job as "running" and
     confirm it is reset to "pending" after the recovery interval.
  7. Smoke-test the Python, Go, and JavaScript client SDKs.

Usage:
    ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=secret python verify.py
    python verify.py --driver          # driver-focused tests only
    python verify.py --client-py       # Python client smoke only
    python verify.py --client-js       # JS client smoke only
    python verify.py --client-go       # Go client smoke only

The script exits 0 on success, 1 on failure.
"""

import argparse, os, sys, time, requests, json
from datetime import datetime, timezone

HOST  = os.getenv("GO_SERVER_HOST", "127.0.0.1")
PORT  = int(os.getenv("GO_SERVER_PORT", "8090"))
BASE  = f"http://{HOST}:{PORT}"

ADMIN_EMAIL    = os.getenv("ADMIN_EMAIL", "admin@example.com")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "supersecretpassword1234")

MAX_WAIT_SECS = 120   # give the driver up to 2 minutes to finish all jobs
TEST_API_TOKEN = "test-api-token-abc-123"
TEST_USER_EMAIL = "user@example.com"

s = requests.Session()


def admin_auth():
    resp = s.post(f"{BASE}/api/collections/_superusers/auth-with-password",
                  json={"identity": ADMIN_EMAIL, "password": ADMIN_PASSWORD})
    resp.raise_for_status()
    token = resp.json()["token"]
    s.headers["Authorization"] = token
    print("[verify] Admin authenticated")


def get_all_jobs():
    resp = s.get(f"{BASE}/api/collections/quantum_jobs/records",
                 params={"perPage": 200, "sort": "+created"})
    resp.raise_for_status()
    return resp.json()["items"]


def get_test_user():
    """Fetch the test user record by email."""
    resp = s.get(f"{BASE}/api/collections/users/records",
                 params={"filter": f'email="{TEST_USER_EMAIL}"'})
    resp.raise_for_status()
    items = resp.json()["items"]
    return items[0] if items else None


def wait_for_completion(timeout=MAX_WAIT_SECS):
    print(f"\n[verify] Waiting up to {timeout}s for all jobs to complete …")
    start = time.time()
    while time.time() - start < timeout:
        jobs = get_all_jobs()
        statuses = [j["status"] for j in jobs]
        pending  = statuses.count("pending")
        running  = statuses.count("running")
        done     = statuses.count("completed")
        total    = len(jobs)
        print(f"  [{int(time.time()-start):3d}s]  total={total}  "
              f"pending={pending}  running={running}  completed={done}")
        if pending == 0 and running == 0 and done == total:
            return jobs
        time.sleep(5)
    return None


def print_summary(jobs):
    print("\n── Job Summary ─────────────────────────────────────────────────")
    fmt = "{:<26} {:<12} {:<20} {}"
    print(fmt.format("ID", "Status", "Finished At", "Results (excerpt)"))
    print("─" * 90)
    for j in jobs:
        results = j.get("results") or "{}"
        if isinstance(results, str):
            try:
                results = json.loads(results)
            except Exception:
                pass
        excerpt = str(results)[:50] + "…" if len(str(results)) > 50 else str(results)
        print(fmt.format(j["id"][:24], j["status"], j.get("finished_at", "")[:19], excerpt))


# ---------------------------------------------------------------------------
# New E2E tests for recently added features
# ---------------------------------------------------------------------------

def test_qpu_seconds_deduction():
    """Verify that QPU seconds were deducted from the test user after jobs ran."""
    print("\n[verify] Testing QPU seconds deduction …")
    user = get_test_user()
    if not user:
        print("[verify] ✗ Test user not found")
        return False

    qpu_seconds = user.get("qpu_seconds", 0)
    # User was granted 1000 seconds; after 5 jobs some should have been deducted
    if qpu_seconds >= 1000:
        print(f"[verify] ✗ QPU seconds not deducted (still {qpu_seconds})")
        return False

    print(f"[verify] ✓ QPU seconds deducted: {qpu_seconds:.2f} remaining")
    return True


def test_api_token_auth():
    """Verify that API token authentication works for job CRUD endpoints."""
    print("\n[verify] Testing API token authentication …")
    headers = {"X-API-Token": TEST_API_TOKEN}

    # List jobs
    resp = requests.get(f"{BASE}/api/jobs", headers=headers)
    if resp.status_code != 200:
        print(f"[verify] ✗ LIST /api/jobs failed: {resp.status_code} {resp.text}")
        return False
    jobs = resp.json()
    print(f"[verify] ✓ LIST /api/jobs returned {len(jobs)} jobs")

    if not jobs:
        print("[verify] ⚠ No jobs to test GET/CANCEL")
        return True

    job_id = jobs[0]["id"]

    # Get single job
    resp = requests.get(f"{BASE}/api/jobs/{job_id}", headers=headers)
    if resp.status_code != 200:
        print(f"[verify] ✗ GET /api/jobs/{job_id} failed: {resp.status_code}")
        return False
    print(f"[verify] ✓ GET /api/jobs/{job_id} succeeded")

    return True


def test_admin_user_update():
    """Verify the admin-only PATCH /api/admin/users/{id} endpoint."""
    print("\n[verify] Testing admin user update endpoint …")
    user = get_test_user()
    if not user:
        print("[verify] ✗ Test user not found")
        return False

    user_id = user["id"]
    original_seconds = user.get("qpu_seconds", 0)

    # Grant additional QPU seconds
    new_seconds = original_seconds + 500.0
    resp = s.patch(f"{BASE}/api/admin/users/{user_id}", json={
        "qpu_seconds": new_seconds,
    })
    if resp.status_code != 200:
        print(f"[verify] ✗ Admin PATCH failed: {resp.status_code} {resp.text}")
        return False

    data = resp.json()
    if abs(data.get("qpu_seconds", 0) - new_seconds) > 0.01:
        print(f"[verify] ✗ qpu_seconds not updated correctly: {data}")
        return False

    print(f"[verify] ✓ Admin PATCH updated qpu_seconds to {data['qpu_seconds']}")
    return True


def test_job_cancel():
    """Test that a pending job can be cancelled via the API."""
    print("\n[verify] Testing job cancellation …")
    headers = {"X-API-Token": TEST_API_TOKEN}

    # Get an existing job to extract the qpu_target (avoids 503 when no online QPU)
    jobs = get_all_jobs()
    qpu_target = jobs[0].get("qpu_target") if jobs else None

    # Submit a new job specifically to cancel it
    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[1];
creg c[1];
x q[0];
measure q[0] -> c[0];"""
    payload = {
        "circuits": [{"circuit": qasm}],
        "shots": 100,
    }
    if qpu_target:
        payload["qpu_target"] = qpu_target
    resp = requests.post(f"{BASE}/api/jobs", json=payload, headers=headers)
    if resp.status_code not in (200, 201):
        print(f"[verify] ✗ Could not create job to cancel: {resp.status_code}")
        return False

    job_id = resp.json().get("job_id") or resp.json().get("id")
    print(f"[verify] Created job {job_id} to cancel")

    # Cancel it
    resp = requests.post(f"{BASE}/api/jobs/{job_id}/cancel", headers=headers)
    if resp.status_code != 200:
        print(f"[verify] ✗ Cancel failed: {resp.status_code} {resp.text}")
        return False

    data = resp.json()
    if data.get("status") != "cancelled":
        print(f"[verify] ✗ Job status not cancelled: {data}")
        return False

    print(f"[verify] ✓ Job {job_id} cancelled successfully")
    return True


def test_python_client_smoke():
    """Smoke-test the Python client SDK against the running server."""
    print("\n[verify] Testing Python client SDK smoke …")
    from qpi_client import QPIClient

    client = QPIClient(BASE, api_token=TEST_API_TOKEN)

    # List jobs
    jobs = client.list_jobs()
    print(f"[verify] ✓ Python client list_jobs returned {len(jobs)} jobs")

    # Get first job if any
    if jobs:
        job = client.get_job(jobs[0]["id"])
        print(f"[verify] ✓ Python client get_job returned status={job.get('status')}")

    client.close()
    return True


def test_go_client_smoke():
    """Smoke-test the Go client SDK against the running server."""
    print("\n[verify] Testing Go client SDK smoke …")
    import subprocess

    script_dir = os.path.dirname(os.path.abspath(__file__))
    env = os.environ.copy()
    env["QPI_BASE_URL"] = BASE
    env["QPI_API_TOKEN"] = TEST_API_TOKEN

    result = subprocess.run(
        ["go", "run", "smoke_go.go"],
        cwd=script_dir,
        env=env,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(f"[verify] ✗ Go client smoke test failed:\n{result.stderr}")
        return False

    for line in result.stdout.strip().splitlines():
        print(f"[verify]   {line}")
    return True


def test_js_client_smoke():
    """Smoke-test the JavaScript client SDK against the running server."""
    print("\n[verify] Testing JS client SDK smoke …")
    import subprocess

    script_dir = os.path.dirname(os.path.abspath(__file__))
    env = os.environ.copy()
    env["QPI_BASE_URL"] = BASE
    env["QPI_API_TOKEN"] = TEST_API_TOKEN

    result = subprocess.run(
        ["node", "smoke_js.mjs"],
        cwd=script_dir,
        env=env,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(f"[verify] ✗ JS client smoke test failed:\n{result.stderr}")
        return False

    for line in result.stdout.strip().splitlines():
        print(f"[verify]   {line}")
    return True


def test_qiskit_hadamard_circuit():
    """Build a Hadamard circuit with Qiskit, submit it, and verify completion."""
    print("\n[verify] Testing Qiskit Hadamard circuit submission …")
    try:
        from qiskit import QuantumCircuit, qasm2
    except ImportError:
        print("[verify] ⚠ qiskit not installed — skipping Hadamard circuit test")
        return True

    # Build a simple Hadamard + measure circuit
    qc = QuantumCircuit(1, 1)
    qc.h(0)
    qc.measure(0, 0)
    qasm = qasm2.dumps(qc)

    # Extract qpu_target from existing jobs so submission succeeds
    # even if the QPU has gone offline after processing seeded jobs
    jobs = get_all_jobs()
    qpu_target = jobs[0].get("qpu_target") if jobs else None

    headers = {"X-API-Token": TEST_API_TOKEN}
    payload = {
        "circuits": [{"circuit": qasm}],
        "shots": 100,
    }
    if qpu_target:
        payload["qpu_target"] = qpu_target

    # Submit the job
    resp = requests.post(f"{BASE}/api/jobs", json=payload, headers=headers)
    if resp.status_code not in (200, 201):
        print(f"[verify] ✗ Job submission failed: {resp.status_code} {resp.text}")
        return False

    job_id = resp.json().get("job_id") or resp.json().get("id")
    print(f"[verify] Submitted Hadamard job {job_id}")

    # Poll for completion (up to 30s)
    for i in range(30):
        time.sleep(1)
        resp = requests.get(f"{BASE}/api/jobs/{job_id}", headers=headers)
        if resp.status_code == 200:
            status = resp.json().get("status")
            if status == "completed":
                print(f"[verify] ✓ Hadamard job completed after {i + 1}s")
                return True
            if status in ("failed", "cancelled"):
                print(f"[verify] ✗ Hadamard job {status}")
                return False

    print("[verify] ✗ Hadamard job did not complete within 30s")
    return False


def test_recovery_engine():
    print("\n[verify] Testing recovery engine …")
    jobs = get_all_jobs()
    if not jobs:
        print("[verify] No jobs found — skipping recovery test")
        return True

    target = jobs[0]
    jid    = target["id"]

    resp = s.patch(f"{BASE}/api/collections/quantum_jobs/records/{jid}",
                   json={"status": "running", "qpu_target": None})
    resp.raise_for_status()
    print(f"[verify] Marked job {jid[:12]}… as 'running'. "
          f"Waiting up to 35s for recovery engine to reset it …")

    for i in range(35):
        time.sleep(1)
        job = s.get(f"{BASE}/api/collections/quantum_jobs/records/{jid}").json()
        if job.get("status") == "pending":
            print(f"[verify] ✓ Recovery engine reset job after {i+1}s")
            return True

    print("[verify] ✗ Recovery engine did NOT reset job within 35s "
          "(may need longer if jobTimeout > 20s in qpi-interface/main.go)")
    return False


def run_driver_tests():
    """Run tests that exercise the driver + core API (no client SDKs)."""
    admin_auth()
    jobs = wait_for_completion()

    if jobs is None:
        print("\n[verify] ✗ FAILED — not all jobs completed within timeout")
        return False

    print(f"\n[verify] ✓ All {len(jobs)} jobs completed!")
    print_summary(jobs)

    all_passed = True

    if not test_qpu_seconds_deduction():
        all_passed = False

    if not test_api_token_auth():
        all_passed = False

    if not test_admin_user_update():
        all_passed = False

    if not test_job_cancel():
        all_passed = False

    if not test_qiskit_hadamard_circuit():
        all_passed = False

    if not test_recovery_engine():
        all_passed = False

    return all_passed


def main():
    parser = argparse.ArgumentParser(description="QPi E2E verification script")
    parser.add_argument("--driver", action="store_true", help="Run driver-focused tests only")
    parser.add_argument("--client-py", action="store_true", help="Run Python client smoke test only")
    parser.add_argument("--client-js", action="store_true", help="Run JS client smoke test only")
    parser.add_argument("--client-go", action="store_true", help="Run Go client smoke test only")
    args = parser.parse_args()

    # If no specific subset requested, run everything
    run_all = not (args.driver or args.client_py or args.client_js or args.client_go)

    all_passed = True

    if run_all or args.driver:
        if not run_driver_tests():
            all_passed = False

    if run_all or args.client_py:
        if not test_python_client_smoke():
            all_passed = False

    if run_all or args.client_go:
        if not test_go_client_smoke():
            all_passed = False

    if run_all or args.client_js:
        if not test_js_client_smoke():
            all_passed = False

    if not all_passed:
        print("\n[verify] ✗ FAILED — one or more checks failed")
        sys.exit(1)

    print("\n[verify] ✓ All checks passed")


if __name__ == "__main__":
    main()
