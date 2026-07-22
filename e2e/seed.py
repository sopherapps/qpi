"""
seed.py — Seeds PocketBase with:
  - A QPU record (with a known access token)
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

HOST = os.getenv("GO_SERVER_HOST", "127.0.0.1")
PORT = int(os.getenv("GO_SERVER_PORT", "8090"))
BASE = f"http://{HOST}:{PORT}"

ADMIN_EMAIL = os.getenv("ADMIN_EMAIL", "admin@example.com")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "supersecretpassword1234")

ACCESS_TOKEN = "my-super-secret-token-12345"
TEST_API_TOKEN = "test-api-token-abc-123"

# Driver-framework mode (RFC 0001): when enabled, also register a driver against
# the QPU and write its one-time token out for the harness to start it with.
DRIVER_FRAMEWORK = os.getenv("QPI_DRIVER_FRAMEWORK", "0") == "1"
DRIVER_KIND = os.getenv("DRIVER_KIND", "mock")
DRIVER_TOKEN_FILE = os.getenv("DRIVER_TOKEN_FILE", "")

s = requests.Session()


def admin_auth():
    resp = s.post(
        f"{BASE}/api/collections/_superusers/auth-with-password",
        json={"identity": ADMIN_EMAIL, "password": ADMIN_PASSWORD},
    )
    resp.raise_for_status()
    token = resp.json()["token"]
    s.headers["Authorization"] = token
    print("[seed] Admin authenticated")


def create_qpu():
    resp = s.post(
        f"{BASE}/api/collections/qpus/records",
        json={
            "name": "qpu_sim_01",
            "access_token": ACCESS_TOKEN,
            "status": "offline",
            "nng_command_port": 0,
            "nng_result_port": 0,
            "enabled": True,
        },
    )
    resp.raise_for_status()
    qpu = resp.json()
    print(f"[seed] QPU created: {qpu['id']}")
    return qpu


def create_driver(qpu_id, kind):
    """Register a driver against the QPU and return its one-time token (RFC 0001 §3)."""
    resp = s.post(
        f"{BASE}/api/op/drivers/create",
        json={
            "name": "driver_sim_01",
            "qpu": qpu_id,
            "kind": kind,
            "language": "python",
        },
    )
    resp.raise_for_status()
    driver = resp.json()
    print(f"[seed] Driver created: {driver['id']} (kind={kind})")
    return driver


def create_user(email="user@example.com", password="userpassword1234"):
    resp = s.post(
        f"{BASE}/api/collections/users/records",
        json={
            "email": email,
            "emailVisibility": True,
            "password": password,
            "passwordConfirm": password,
        },
    )
    if resp.status_code == 400 and "validation_not_unique" in resp.text.lower() :
       # fetch existing
        resp2 = s.get(
            f"{BASE}/api/collections/users/records",
            params={"filter": f'email="{email}"'},
        )
        resp2.raise_for_status()
        user = resp2.json()["items"][0]
        print(f"[seed] User already exists: {user['id']}")
        return user
    if resp.status_code >= 400:
        print(f"[seed] User creation failed: {resp.status_code} {resp.text}")
    resp.raise_for_status()
    user = resp.json()
    print(f"[seed] User created: {user['id']}")
    return user


def grant_user_qpu_time(user_id, qpu_seconds=1000.0, api_tokens=None):
    """Grant QPU seconds and seed API tokens via collection endpoints."""
    if api_tokens is None:
        api_tokens = [TEST_API_TOKEN]

    # 1. Update qpu_seconds on the user record
    resp = s.patch(
        f"{BASE}/api/collections/users/records/{user_id}",
        json={
            "qpu_seconds": qpu_seconds,
        },
    )
    resp.raise_for_status()
    user_data = resp.json()

    # 2. Seed API tokens in the api_tokens collection
    import hashlib

    created_tokens = []
    for token_value in api_tokens:
        hashed = hashlib.sha256(token_value.encode()).hexdigest()

        # Check if this token record already exists
        filter_str = f'token="{hashed}" && user="{user_id}"'
        resp_check = s.get(
            f"{BASE}/api/collections/api_tokens/records", params={"filter": filter_str}
        )
        resp_check.raise_for_status()
        items = resp_check.json().get("items", [])
        if items:
            created_tokens.append(items[0])
            continue

        # Create new token record
        resp_create = s.post(
            f"{BASE}/api/collections/api_tokens/records",
            json={
                "token": hashed,
                "user": user_id,
                "name": "seeded-token",
            },
        )
        resp_create.raise_for_status()
        created_tokens.append(resp_create.json())

    print(
        f"[seed] Granted {user_data.get('qpu_seconds')} qpu_seconds and seeded {len(created_tokens)} api_tokens for user {user_id}"
    )
    return {
        "id": user_id,
        "qpu_seconds": user_data.get("qpu_seconds"),
        "api_tokens": created_tokens,
    }


def create_time_slot(user_id):
    now = datetime.now(timezone.utc)
    start = (now - timedelta(minutes=2)).strftime("%Y-%m-%d %H:%M:%S.000Z")
    end = (now + timedelta(minutes=5)).strftime("%Y-%m-%d %H:%M:%S.000Z")
    resp = s.post(
        f"{BASE}/api/collections/time_slots/records",
        json={
            "start_time": start,
            "end_time": end,
            "booked_by": user_id,
        },
    )
    if resp.status_code >= 400:
        print(f"[seed] Time slot creation failed: {resp.status_code} {resp.text}")
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
        job_ids.append(job["id"])
        print(f"[seed] Job {i + 1}/{n} created via API: {job['id']}")
    return job_ids


if __name__ == "__main__":
    admin_auth()
    qpu = create_qpu()

    if DRIVER_FRAMEWORK:
        driver = create_driver(qpu["id"], DRIVER_KIND)
        if DRIVER_TOKEN_FILE:
            with open(DRIVER_TOKEN_FILE, "w") as f:
                f.write(driver.get("token", ""))
            print(f"[seed] Wrote driver token to {DRIVER_TOKEN_FILE}")

    user = create_user()
    grant_user_qpu_time(user["id"], qpu_seconds=1000.0, api_tokens=[TEST_API_TOKEN])
    create_time_slot(user["id"])
    create_jobs_via_api(user["id"], qpu["id"], n=5)
    
    empty_user = create_user("emptyuser@example.com", "userpassword1234")
    grant_user_qpu_time(empty_user["id"], qpu_seconds=1000.0, api_tokens=[])

    print(f"\n[seed] Done!  ACCESS_TOKEN={ACCESS_TOKEN}")
    print(
        f"[seed] Run the driver with:  QPI_ACCESS_TOKEN={ACCESS_TOKEN} python driver.py"
    )
