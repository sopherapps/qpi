<pre align="center">
 ██████╗ ██████╗ ██╗
██╔═══██╗██╔══██╗██║
██║   ██║██████╔╝██║
██║▄▄ ██║██╔═══╝ ██║
╚██████╔╝██║     ██║
 ╚══▀▀═╝ ╚═╝     ╚═╝
</pre>

<h1 align="center">QPI QPU Driver</h1>

<p align="center">
  <a href="https://badge.fury.io/py/qpi-driver"><img src="https://badge.fury.io/py/qpi-driver.svg" alt="PyPI version"></a>
  <a href="https://github.com/sopherapps/qpi/actions/workflows/ci.yml"><img src="https://github.com/sopherapps/qpi/actions/workflows/ci.yml/badge.svg" alt="CI/CD Workflow"></a>
  <a href="https://github.com/sopherapps/qpi/releases"><img src="https://img.shields.io/github/v/tag/sopherapps/qpi?label=version" alt="GitHub Tag"></a>
  <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
</p>

<p align="center">
  Python QPU driver for the <a href="https://github.com/sopherapps/qpi">QPI</a> quantum computing platform.
  Runs on isolated hardware nodes controlling the QPU via multiple executor backends.
</p>

<p align="center">
  <strong><a href="https://sopherapps.github.io/qpi/driver/">📚 Read the Documentation</a></strong>
</p>

---

## Install

### Base package (mock executor)

```bash
pip install qpi-driver
```

### With CLI support

```bash
pip install "qpi-driver[cli]"
```

### With Qiskit Aer simulator

```bash
pip install "qpi-driver[aer]"
```

### With Quantify/Qblox hardware support

```bash
pip install "qpi-driver[quantify]"
```

Requires **Python ≥ 3.12, < 3.13**.

---

## Quick Start

### CLI

```bash
# Connect a mock QPU to the server
qpi-driver start \
  --qpi-addr http://localhost:8090 \
  --token <qpu-access-token> \
  --ca-fingerprint <fingerprint> \
  --name qpu_sim_01 \
  --executor mock \
  --data-dir ./data
```

Environment variables are also supported:

```bash
export QPI_ADDR=http://localhost:8090
export QPI_ACCESS_TOKEN=<token>
export QPI_CA_FINGERPRINT=<fingerprint>
export QPU_NAME=qpu_sim_01
export DRIVER_BACKEND=mock
qpi-driver start
```

### systemd Service (Linux)

To run `qpi-driver` persistently on a Linux machine in the background, you can use `systemd`. 

We have provided a standalone interactive bash installer that automates the entire process (installing `uv`, installing the `qpi-driver` tool, prompting for your tokens/addresses, and registering the systemd service):

```bash
# Run the interactive systemd installer script directly via curl
sudo bash -c "$(curl -LsSf https://raw.githubusercontent.com/sopherapps/qpi/main/qpi-driver/install-systemd.sh)"
```

Alternatively, you can run the installer non-interactively by specifying all environment variables:
```bash
curl -LsSf https://raw.githubusercontent.com/sopherapps/qpi/main/qpi-driver/install-systemd.sh | sudo \
  QPI_TOKEN="<your-qpi-access-token>" \
  QPI_ADDR="http://127.0.0.1:8090" \
  CA_FINGERPRINT="<fingerprint>" \
  QPU_NAME="rigetti-aspen-1" \
  EXECUTOR="qblox" \
  bash
```

#### Manual systemd Installation
If you prefer to configure it manually, follow these steps:

1. **Install `uv`** (a fast Python package installer):
   ```bash
   curl -LsSf https://astral.sh/uv/install.sh | sh
   source $HOME/.local/bin/env
   ```

2. **Install `qpi-driver` as a tool**:
   Make sure to specify the correct extras (e.g. `[cli,qblox]`, `[cli,aer]`):
   ```bash
   uv tool install "qpi-driver[cli,qblox]"
   ```

3. **Create the systemd unit file**:
   Replace the placeholder `<values>` with your actual configuration.
   ```bash
   sudo bash -c 'cat > /etc/systemd/system/rigetti-aspen-1.qpi-driver.service <<EOF
   [Unit]
   Description=QPI Driver Service (rigetti-aspen-1)
   After=network.target

   [Service]
   Type=simple

   Environment="QPI_ACCESS_TOKEN=<your-qpi-access-token>"
   Environment="QPI_DATA_DIR=/var/qpi-driver/rigetti-aspen-1"
   Environment="QPI_CA_FILE=/var/qpi-driver/rigetti-aspen-1/qpi.ca.pem"
   Environment="QPI_QUANTIFY_DEVICE_CONFIG=/var/qpi-driver/rigetti-aspen-1/quantify.device.yml"
   Environment="QPI_QUANTIFY_HARDWARE_CONFIG=/var/qpi-driver/rigetti-aspen-1/quantify.hardware.json"
   Environment=PYTHONUNBUFFERED=1

   ExecStart=/home/<user>/.local/bin/qpi-driver start \
           --ca-fingerprint <your-fingerprint> \
           --qpi-addr <your-qpi-server-address> \
           --name "rigetti-aspen-1" \
           --executor "qblox"

   Restart=on-failure
   User=<user>

   StandardOutput=journal
   StandardError=journal
   SyslogIdentifier=rigetti-aspen-1.qpi-driver

   [Install]
   WantedBy=multi-user.target
   EOF'
   ```

4. **Start and enable the service**:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable rigetti-aspen-1.qpi-driver.service
   sudo systemctl start rigetti-aspen-1.qpi-driver.service
   sudo systemctl status rigetti-aspen-1.qpi-driver.service
   ```

### Python API

```python
from qpi_driver import run_driver

run_driver(
    qpi_addr="http://localhost:8090",
    token="<qpu-access-token>",
    ca_fingerprint="<fingerprint>",
    name="qpu_sim_01",
    executor="mock",
    data_dir="./data",
)
```

### Custom executor

```python
from qpi_driver import Executor, run_driver
from qpi_driver.executors.mock import MockExecutor

class MyCustomExecutor(Executor):
    def execute(self, payload):
        # Your QPU-specific execution logic
        ...

run_driver(
    qpi_addr="http://localhost:8090",
    token="<token>",
    ca_fingerprint="<fingerprint>",
    name="my_qpu",
    executor="custom",
    custom_executor=MyCustomExecutor(),
)
```

---

## Executor Backends

| Backend | Description | Extra |
|---------|-------------|-------|
| `mock` | Qiskit BasicSimulator (default) | — |
| `qiskit_aer` | Qiskit Aer simulator | `[aer]` |
| `quantify` | Quantify-scheduler + Qblox instruments | `[quantify]` |
| `qblox` | Qblox scheduler (legacy) | `[qblox]` |

---

## Architecture

The driver uses Python's `multiprocessing` library to isolate responsibilities:

- **Main Process**: NNG PULL listener, receives commands from server
- **Worker Process**: Executes quantum circuits via the configured executor
- **Result Sender Process**: NNG PUSH, sends results back to server

```
┌─────────────┐     NNG PUSH      ┌─────────────────┐
│ Server│ ────────────────> │  Main Process   │
│  Dispatcher │                   │  (PULL listener)│
└─────────────┘                   └────────┬────────┘
                                           │
                              multiprocessing.Queue
                                           │
                                           ▼
                                   ┌───────────────┐
                                   │ Worker Process│
                                   │  (Executor)   │
                                   └───────┬───────┘
                                           │
                              multiprocessing.Queue
                                           │
                                           ▼
                                   ┌───────────────┐
                                   │Result Sender  │
                                   │  (PUSH)       │
                                   └───────┬───────┘
                                           │ NNG PUSH
                                           ▼
                                   ┌───────────────┐
                                   │  Server │
                                   │   Listener    │
                                   └───────────────┘
```

---

## CLI Reference

```
qpi-driver start [OPTIONS]

Options:
  -a, --qpi-addr TEXT          QPI server URL [env: QPI_ADDR]
  -t, --token TEXT             QPU access token [env: QPI_ACCESS_TOKEN]
  -n, --name TEXT              QPU name [env: QPU_NAME]
  -e, --executor TEXT          Backend: mock, qiskit_aer, quantify, qblox, presto [env: DRIVER_BACKEND]
  -d, --data-dir PATH          Data directory [env: QPI_DATA_DIR]
  --is-dummy                   Run in dummy/simulation mode
  --quantify-hardware-config PATH  Quantify hardware config [env: QPI_QUANTIFY_HARDWARE_CONFIG]
  --quantify-device-config PATH    Quantify device config [env: QPI_QUANTIFY_DEVICE_CONFIG]
  --job-timeout INTEGER        Job timeout in seconds [env: QPI_JOB_TIMEOUT]
  --ca-file PATH               Path to the CA root certificate [env: QPI_CA_FILE]
  --ca-fingerprint TEXT        Fingerprint to verify the CA root certificate [env: QPI_CA_FINGERPRINT]
  --help                       Show this message and exit.
```

---

## Documentation

- [Main QPI Repository](https://github.com/sopherapps/qpi)
- [PyPI Project Page](https://pypi.org/project/qpi-driver/)

---

## License

MIT — see the [main repository](https://github.com/sopherapps/qpi/blob/main/LICENSE) for details.
