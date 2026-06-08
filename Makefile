.PHONY: all build test lint lint-go lint-py format format-go format-py package clean venv-check

VERSION ?= 1.0.0
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
	$(UV) sync --project qpi-driver --extra cli --extra aer --extra quantify --extra test

test:
	@echo "Running Go unit tests..."
	(cd qpi-interface && go test -v ./...)
	@echo "Running Python unit tests..."
	$(UV) run --project qpi-driver pytest qpi-driver/tests/
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
