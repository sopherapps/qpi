"""
verify.py — End-to-end verification of the QPi control stack.

Steps:
  1. Poll quantum_jobs until all seeded jobs are "completed".
  2. Print a summary table of each job's status and results.
  3. Test the recovery engine: manually mark one job as "running" and
     confirm it is reset to "pending" after the recovery interval.

Usage:
    ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=secret python verify.py

The script exits 0 on success, 1 on failure.
"""

import os, sys, time, requests, json
from datetime import datetime, timezone

HOST  = os.getenv("GO_SERVER_HOST", "127.0.0.1")
PORT  = int(os.getenv("GO_SERVER_PORT", "8090"))
BASE  = f"http://{HOST}:{PORT}"

ADMIN_EMAIL    = os.getenv("ADMIN_EMAIL", "admin@example.com")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "supersecretpassword1234")

MAX_WAIT_SECS = 120   # give the driver up to 2 minutes to finish all jobs

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
          "(may need longer if jobTimeout > 20s in main.go)")
    return False


def main():
    admin_auth()
    jobs = wait_for_completion()

    if jobs is None:
        print("\n[verify] ✗ FAILED — not all jobs completed within timeout")
        sys.exit(1)

    print(f"\n[verify] ✓ All {len(jobs)} jobs completed!")
    print_summary(jobs)
    if not test_recovery_engine():
        print("\n[verify] ✗ FAILED — recovery engine test failed")
        sys.exit(1)
    print("\n[verify] ✓ All checks passed")


if __name__ == "__main__":
    main()
