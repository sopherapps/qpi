<pre align="center">
 ██████╗ ██████╗ ██╗
██╔═══██╗██╔══██╗██║
██║   ██║██████╔╝██║
██║▄▄ ██║██╔═══╝ ██║
╚██████╔╝██║     ██║
 ╚══▀▀═╝ ╚═╝     ╚═╝
</pre>

<h1 align="center">QPI Hardware Driver</h1>

<p align="center">
  <a href="https://badge.fury.io/py/qpi-driver"><img src="https://badge.fury.io/py/qpi-driver.svg" alt="PyPI version"></a>
  <a href="https://github.com/sopherapps/qpi/actions/workflows/ci.yml"><img src="https://github.com/sopherapps/qpi/actions/workflows/ci.yml/badge.svg" alt="CI/CD Workflow"></a>
  <a href="https://github.com/sopherapps/qpi/releases"><img src="https://img.shields.io/github/v/tag/sopherapps/qpi?label=version" alt="GitHub Tag"></a>
  <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
</p>

<p align="center">
  Python hardware driver for the <a href="https://github.com/sopherapps/qpi">QPI</a> quantum computing platform.
  Runs on isolated hardware nodes controlling the QPU via multiple executor backends.
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
# Connect a mock QPU to the orchestrator
qpi-driver start \
  --qpi-addr http://localhost:8090 \
  --token <qpu-access-token> \
  --name qpu_sim_01 \
  --executor mock \
  --data-dir ./data
```

Environment variables are also supported:

```bash
export QPI_ADDR=http://localhost:8090
export QPI_ACCESS_TOKEN=<token>
export QPU_NAME=qpu_sim_01
export DRIVER_BACKEND=mock
qpi-driver start
```

### Python API

```python
from qpi_driver import run_driver

run_driver(
    qpi_addr="http://localhost:8090",
    token="<qpu-access-token>",
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
| `presto` | Presto RF control | `[quantify]` |

---

## Architecture

The driver uses Python's `multiprocessing` library to isolate responsibilities:

- **Main Process**: NNG PULL listener, receives commands from orchestrator
- **Worker Process**: Executes quantum circuits via the configured executor
- **Result Sender Process**: NNG PUSH, sends results back to orchestrator

```
┌─────────────┐     NNG PUSH      ┌─────────────────┐
│ Orchestrator│ ────────────────> │  Main Process   │
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
                                   │  Orchestrator │
                                   │   Listener    │
                                   └───────────────┘
```

---

## CLI Reference

```
qpi-driver start [OPTIONS]

Options:
  -a, --qpi-addr TEXT          QPI orchestrator URL [env: QPI_ADDR]
  -t, --token TEXT             QPU access token [env: QPI_ACCESS_TOKEN]
  -n, --name TEXT              QPU name [env: QPU_NAME]
  -e, --executor TEXT          Backend: mock, qiskit_aer, quantify, qblox, presto [env: DRIVER_BACKEND]
  -d, --data-dir PATH          Data directory [env: QPI_DATA_DIR]
  --is-dummy                   Run in dummy/simulation mode
  --quantify-hardware-config PATH  Quantify hardware config [env: QPI_QUANTIFY_HARDWARE_CONFIG]
  --help                       Show this message and exit.
```

---

## Documentation

- [Main QPI Repository](https://github.com/sopherapps/qpi)
- [PyPI Project Page](https://pypi.org/project/qpi-driver/)

---

## License

MIT — see the [main repository](https://github.com/sopherapps/qpi/blob/main/LICENSE) for details.
