"""
seed.py — Seeds PocketBase with:
  - A QPU record (with a known registration token)
  - A test user (with 0 qpu_seconds, then granted time via admin API)
  - API tokens assigned to the test user via admin API
  - A time slot (active for the next 5 minutes)
  - Several pending quantum jobs submitted via the custom REST API

Usage:
    ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=secret python seed.py

Env vars:
    GO_SERVER_HOST   (default: 127.0.0.1)
    GO_SERVER_PORT   (default: 8090)
    ADMIN_EMAIL
    ADMIN_PASSWORD
"""

import os
import requests
from datetime import datetime, timezone, timedelta

HOST  = os.getenv("GO_SERVER_HOST", "127.0.0.1")
PORT  = int(os.getenv("GO_SERVER_PORT", "8090"))
BASE  = f"http://{HOST}:{PORT}"

ADMIN_EMAIL    = os.getenv("ADMIN_EMAIL", "admin@example.com")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "supersecretpassword1234")

REGISTRATION_TOKEN = "my-super-secret-token-12345"
TEST_API_TOKEN     = "test-api-token-abc-123"

s = requests.Session()


def admin_auth():
    resp = s.post(f"{BASE}/api/collections/_superusers/auth-with-password",
                  json={"identity": ADMIN_EMAIL, "password": ADMIN_PASSWORD})
    resp.raise_for_status()
    token = resp.json()["token"]
    s.headers["Authorization"] = token
    print("[seed] Admin authenticated")


def create_qpu():
    resp = s.post(f"{BASE}/api/collections/qpus/records", json={
        "name": "QPU-Sim-01",
        "registration_token": REGISTRATION_TOKEN,
        "status": "offline",
        "nng_command_port": 0,
        "nng_result_port": 0,
        "enabled": True,
    })
    resp.raise_for_status()
    qpu = resp.json()
    print(f"[seed] QPU created: {qpu['id']}")
    return qpu


def create_user(email="user@example.com", password="userpassword1234"):
    resp = s.post(f"{BASE}/api/collections/users/records", json={
        "email": email,
        "emailVisibility": True,
        "password": password,
        "passwordConfirm": password,
    })
    if resp.status_code == 400 and "already" in resp.text.lower():
        # fetch existing
        resp2 = s.get(f"{BASE}/api/collections/users/records",
                      params={"filter": f'email="{email}"'})
        resp2.raise_for_status()
        user = resp2.json()["items"][0]
        print(f"[seed] User already exists: {user['id']}")
        return user
    resp.raise_for_status()
    user = resp.json()
    print(f"[seed] User created: {user['id']}")
    return user


def grant_user_qpu_time(user_id, qpu_seconds=1000.0, api_tokens=None):
    """Use the admin-only PATCH endpoint to grant QPU seconds and API tokens."""
    if api_tokens is None:
        api_tokens = [TEST_API_TOKEN]
    resp = s.patch(f"{BASE}/api/admin/users/{user_id}", json={
        "qpu_seconds": qpu_seconds,
        "api_tokens": api_tokens,
    })
    resp.raise_for_status()
    data = resp.json()
    print(f"[seed] Granted {data.get('qpu_seconds')} qpu_seconds and {len(data.get('api_tokens', []))} api_tokens to user {user_id}")
    return data


def create_time_slot(user_id):
    now   = datetime.now(timezone.utc)
    start = (now - timedelta(minutes=2)).strftime("%Y-%m-%d %H:%M:%S.000Z")
    end   = (now + timedelta(minutes=5)).strftime("%Y-%m-%d %H:%M:%S.000Z")
    resp = s.post(f"{BASE}/api/collections/time_slots/records", json={
        "start_time": start,
        "end_time":   end,
        "booked_by":  user_id,
    })
    resp.raise_for_status()
    slot = resp.json()
    print(f"[seed] Time slot created: {slot['id']}  ({start} → {end})")
    return slot


def create_jobs_via_api(user_id, qpu_id, n=5):
    """Submit jobs via the custom /api/jobs REST endpoint using API token auth."""
    job_ids = []
    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
h q[0];
cx q[0], q[1];
measure q -> c;"""

    # Use API token auth for job submission
    headers = {
        "Content-Type": "application/json",
        "X-API-Token": TEST_API_TOKEN,
    }

    for i in range(n):
        payload = {
            "circuits": [{"circuit": qasm}],
            "shots": 1024,
            "meas_level": 2,
            "meas_return": "single",
            "qpu_target": qpu_id,
        }
        resp = requests.post(f"{BASE}/api/jobs", json=payload, headers=headers)
        resp.raise_for_status()
        job = resp.json()
        job_ids.append(job["job_id"])
        print(f"[seed] Job {i+1}/{n} created via API: {job['job_id']}")
    return job_ids


if __name__ == "__main__":
    admin_auth()
    qpu  = create_qpu()
    user = create_user()
    grant_user_qpu_time(user["id"], qpu_seconds=1000.0, api_tokens=[TEST_API_TOKEN])
    create_time_slot(user["id"])
    create_jobs_via_api(user["id"], qpu["id"], n=5)
    print(f"\n[seed] Done!  REGISTRATION_TOKEN={REGISTRATION_TOKEN}")
    print(f"[seed] Run the driver with:  REGISTRATION_TOKEN={REGISTRATION_TOKEN} python driver.py")
