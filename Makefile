.PHONY: all clean tests lint build format coverage benchmarks module-check fixture-module-check fixture-module-tidy

all: build tests lint

MODULES = . hostmock function httpclient kv logging metrics sql
SUBMODULES = hostmock function httpclient kv logging metrics sql
FIXTURE_MODULES = testdata/tinygo

# Run tests for all components
tests:
	@echo "Running tests for all modules..."
	go test -v -race -covermode=atomic -coverprofile=coverage-root.out ./...
	@for dir in $(SUBMODULES); do \
		$(MAKE) -C $$dir tests || exit 1; \
	done
	@echo "Merging coverage reports..."
	@{ \
		echo "mode: atomic"; \
		tail -n +2 coverage-root.out; \
		for dir in $(SUBMODULES); do \
			tail -n +2 "$$dir/coverage.out"; \
		done; \
	} > coverage.out
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out -o coverage.html

coverage: tests

benchmarks:
	@echo "Running benchmarks for all components..."
	@for dir in $(SUBMODULES); do \
		$(MAKE) -C $$dir benchmarks || exit 1; \
	done

# Build all components
build:
	@echo "Building all modules..."
	go build ./...
	@for dir in $(SUBMODULES); do \
		$(MAKE) -C $$dir build || exit 1; \
	done

# Format all code
format:
	@echo "Formatting code..."
	@find . -type f -name "*.go" -not -path "./vendor/*" -print0 | xargs -0 gofmt -s -w
	@find . -type f -name "*.go" -not -path "./vendor/*" -print0 | xargs -0 goimports -w
	@find . -type f -name "*.go" -not -path "./vendor/*" -print0 | xargs -0 golines -m 120 -w
	@for dir in $(SUBMODULES); do \
		$(MAKE) -C $$dir format || exit 1; \
	done

# Lint all code
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		set -e; \
		golangci-lint run ./...; \
		for dir in $(SUBMODULES); do \
			$(MAKE) -C $$dir lint || exit 1; \
		done; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi

# Verify every module works without workspace replacements or go.mod changes.
module-check:
	@for dir in $(MODULES); do \
		echo "Checking module $$dir..."; \
		(cd "$$dir" && GOWORK=off go mod tidy -diff) || exit 1; \
		(cd "$$dir" && GOWORK=off go build -mod=readonly ./...) || exit 1; \
		(cd "$$dir" && GOWORK=off go test -mod=readonly ./...) || exit 1; \
	done

# Verify fixture modules without loading test-only dependencies of published modules.
fixture-module-check:
	@for dir in $(FIXTURE_MODULES); do \
		echo "Checking fixture module $$dir..."; \
		(cd "$$dir" && GOWORK=off go build -mod=readonly ./...) || exit 1; \
		(cd "$$dir" && GOWORK=off go test -mod=readonly ./...) || exit 1; \
	done

# Check fixture metadata with the current Go version, which can load the full published module graph.
fixture-module-tidy:
	@for dir in $(FIXTURE_MODULES); do \
		echo "Checking fixture metadata $$dir..."; \
		(cd "$$dir" && GOWORK=off go mod tidy -diff) || exit 1; \
	done

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@for dir in $(SUBMODULES); do \
		$(MAKE) -C $$dir clean || exit 1; \
	done
	@find . -type f -name "*.test" -delete
	@find . -type f -name "coverage.out" -delete
	@find . -type f -name "coverage.html" -delete
	@rm -f coverage-root.out
	@find . -type d -name "vendor" -exec rm -rf {} + 2>/dev/null || true
	@rm -rf bin/
