.PHONY: all build test lint lint-go lint-py format format-go format-py package clean venv-check

VERSION ?= 0.0.1
UV := $(shell command -v uv 2> /dev/null || echo "$$HOME/.local/bin/uv")

all: build

# Automatically create the virtual environment if not already in one and uv is missing.
venv-check:
	@if ! command -v uv >/dev/null 2>&1 && [ ! -f "$$HOME/.local/bin/uv" ]; then \
		echo "uv not found, installing..."; \
		curl -LsSf https://astral.sh/uv/install.sh | sh; \
	fi

build: venv-check
	@echo "Building Go orchestrator..."
	mkdir -p bin
	(cd qpi-interface && go build -o ../bin/qpi .)
	@echo "Installing python package..."
	$(UV) sync --project qpi-driver --extra cli --extra aer --extra quantify --dev
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "Fixing macOS codesign for q1asm_macos..."; \
		codesign --force --deep --sign - qpi-driver/.venv/lib/python3.12/site-packages/qblox_instruments/assemblers/q1asm_macos 2>/dev/null || true; \
	fi


test: test-go test-py test-e2e

test-go:
	@echo "Running Go unit tests..."
	(cd qpi-interface && go test -v ./...)

test-py: test-py-base test-py-cli test-py-aer test-py-quantify

test-py-base:
	@echo "Running Python tests with base deps only (mock executor)..."
	$(UV) sync --project qpi-driver --dev
	$(UV) run --project qpi-driver pytest qpi-driver/tests/ -v

test-py-cli:
	@echo "Running Python tests with [cli] extra..."
	$(UV) sync --project qpi-driver --extra cli --dev
	$(UV) run --project qpi-driver pytest qpi-driver/tests/ -v

test-py-aer:
	@echo "Running Python tests with [aer] extra..."
	$(UV) sync --project qpi-driver --extra aer --dev
	$(UV) run --project qpi-driver pytest qpi-driver/tests/ -v

test-py-quantify:
	@echo "Running Python tests with [quantify] extra..."
	$(UV) sync --project qpi-driver --extra quantify --dev
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "Fixing macOS codesign for q1asm_macos..."; \
		codesign --force --deep --sign - qpi-driver/.venv/lib/python3.12/site-packages/qblox_instruments/assemblers/q1asm_macos 2>/dev/null || true; \
	fi
	$(UV) run --project qpi-driver pytest qpi-driver/tests/ -v

test-e2e:
	@echo "Running E2E tests..."
	./e2e/run_tests.sh

lint: lint-go lint-py

lint-go:
	@echo "Linting Go files..."
	(cd qpi-interface && go vet ./...)
	(cd qpi-interface && gofmt -l -d .)

lint-py:
	@echo "Linting Python files..."
	$(UV) run --project qpi-driver ruff check qpi-driver/

format: format-go format-py

format-go:
	@echo "Formatting Go files..."
	(cd qpi-interface && go fmt ./...)

format-py:
	@echo "Formatting and sorting imports for Python files..."
	$(UV) run --project qpi-driver ruff format qpi-driver/
	$(UV) run --project qpi-driver ruff check --select I --fix qpi-driver/

package:
	@echo "Packaging Go application..."
	./scripts/package.sh $(VERSION)

clean:
	@echo "Cleaning up..."
	rm -rf bin/pb_data bin/data bin/qpi bin/dist bin/builds
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type d -name "*.egg-info" -exec rm -rf {} +
	find . -type d -name ".pytest_cache" -exec rm -rf {} +
	find . -type d -name ".ruff_cache" -exec rm -rf {} +
	rm -rf qpi-driver/build qpi-driver/dist qpi-driver/*.egg-info qpi-driver/.venv .venv
