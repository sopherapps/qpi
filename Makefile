.PHONY: all build test clean venv-check

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
		.venv/bin/python -m pip install -e ./qpi-driver[cli,aer]; \
	else \
		python -m pip install -e ./qpi-driver[cli,aer]; \
	fi

test:
	@echo "Running E2E tests..."
	./e2e/run_tests.sh

clean:
	@echo "Cleaning up..."
	rm -rf bin/pb_data bin/data bin/qpi
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type d -name "*.egg-info" -exec rm -rf {} +
	find . -type d -name ".pytest_cache" -exec rm -rf {} +
	rm -rf qpi-driver/build qpi-driver/dist qpi-driver/*.egg-info
