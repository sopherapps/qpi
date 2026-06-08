.PHONY: all build test lint lint-go lint-py format format-go format-py package clean venv-check

VERSION ?= 1.0.0

all: build

# Automatically create the virtual environment if not already in one and .venv doesn't exist.
venv-check:
	@if [ -z "$$VIRTUAL_ENV" ] && [ ! -d ".venv" ]; then \
		echo "Creating virtual environment..."; \
		python3 -m venv .venv; \
		.venv/bin/pip install --upgrade pip; \
	fi

build: venv-check
	@echo "Building Go orchestrator..."
	mkdir -p bin
	(cd qpi-interface && go build -o ../bin/qpi .)
	@echo "Installing python package..."
	@if [ -d ".venv" ] && [ -z "$$VIRTUAL_ENV" ]; then \
		.venv/bin/python -m pip install -e ./qpi-driver[cli,aer,test]; \
	else \
		python -m pip install -e ./qpi-driver[cli,aer,test]; \
	fi

test:
	@echo "Running Go unit tests..."
	(cd qpi-interface && go test -v ./...)
	@echo "Running Python unit tests..."
	@if [ -d ".venv" ] && [ -z "$$VIRTUAL_ENV" ]; then \
		.venv/bin/python -m pytest qpi-driver/tests/; \
	else \
		python -m pytest qpi-driver/tests/; \
	fi
	@echo "Running E2E tests..."
	./e2e/run_tests.sh

lint: lint-go lint-py

lint-go:
	@echo "Linting Go files..."
	(cd qpi-interface && go vet ./...)
	(cd qpi-interface && gofmt -l -d .)

lint-py:
	@echo "Linting Python files..."
	@if [ -d ".venv" ] && [ -z "$$VIRTUAL_ENV" ]; then \
		.venv/bin/ruff check qpi-driver/; \
	else \
		ruff check qpi-driver/; \
	fi

format: format-go format-py

format-go:
	@echo "Formatting Go files..."
	(cd qpi-interface && go fmt ./...)

format-py:
	@echo "Formatting and sorting imports for Python files..."
	@if [ -d ".venv" ] && [ -z "$$VIRTUAL_ENV" ]; then \
		.venv/bin/ruff format qpi-driver/; \
		.venv/bin/ruff check --select I --fix qpi-driver/; \
	else \
		ruff format qpi-driver/; \
		ruff check --select I --fix qpi-driver/; \
	fi

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
	rm -rf qpi-driver/build qpi-driver/dist qpi-driver/*.egg-info
