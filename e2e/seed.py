"""
seed.py — Seeds PocketBase with:
  - A QPU record (with a known registration token)
  - A test user
  - A time slot (active for the next 5 minutes)
  - Several pending quantum jobs

Usage:
    ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=secret python seed.py

Env vars:
    GO_SERVER_HOST   (default: 127.0.0.1)
    GO_SERVER_PORT   (default: 8090)
    ADMIN_EMAIL
    ADMIN_PASSWORD
"""

import os, json, requests
from datetime import datetime, timezone, timedelta

HOST  = os.getenv("GO_SERVER_HOST", "127.0.0.1")
PORT  = int(os.getenv("GO_SERVER_PORT", "8090"))
BASE  = f"http://{HOST}:{PORT}"

ADMIN_EMAIL    = os.getenv("ADMIN_EMAIL", "admin@example.com")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "supersecretpassword1234")

REGISTRATION_TOKEN = "my-super-secret-token-12345"

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


def create_jobs(user_id, qpu_id, n=5):
    job_ids = []
    qasm = """OPENQASM 2.0;
include "qelib1.inc";
qreg q[2];
creg c[2];
h q[0];
cx q[0], q[1];
measure q -> c;"""
    for i in range(n):
        payload = {"n_qubits": 2, "shots": 1024, "qasm": qasm}
        resp = s.post(f"{BASE}/api/collections/quantum_jobs/records", json={
            "user_id":    user_id,
            "qpu_target": qpu_id,
            "payload":    json.dumps(payload),
            "status":     "pending",
        })
        resp.raise_for_status()
        job = resp.json()
        job_ids.append(job["id"])
        print(f"[seed] Job {i+1}/{n} created: {job['id']}")
    return job_ids



if __name__ == "__main__":
    admin_auth()
    qpu  = create_qpu()
    user = create_user()
    create_time_slot(user["id"])
    create_jobs(user["id"], qpu["id"], n=5)
    print(f"\n[seed] Done!  REGISTRATION_TOKEN={REGISTRATION_TOKEN}")
    print(f"[seed] Run the driver with:  REGISTRATION_TOKEN={REGISTRATION_TOKEN} python driver.py")
