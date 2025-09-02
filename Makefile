.PHONY: all clean tests lint build format benchmarks

all: build tests lint

COMPONENTS = http

# Run tests for all components
tests:
	@echo "Running tests for all components..."
	@for dir in $(COMPONENTS); do \
		$(MAKE) -C $$dir tests || exit 1; \
	done



benchmarks:
	@echo "Running benchmarks for all components..."
	@for dir in $(COMPONENTS); do \
		$(MAKE) -C $$dir benchmarks || exit 1; \
	done

# Build all components
build:
	@echo "Building all components..."
	@for dir in $(COMPONENTS); do \
		$(MAKE) -C $$dir build || exit 1; \
	done

# Format all code
format:
	@echo "Formatting code..."
	@find . -type f -name "*.go" -not -path "./vendor/*" -print0 | xargs -0 gofmt -s -w
	@find . -type f -name "*.go" -not -path "./vendor/*" -print0 | xargs -0 goimports -w
	@find . -type f -name "*.go" -not -path "./vendor/*" -print0 | xargs -0 golines -m 120 -w
	@for dir in $(COMPONENTS); do \
		$(MAKE) -C $$dir format || exit 1; \
	done

# Lint all code
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		for dir in $(COMPONENTS); do \
			$(MAKE) -C $$dir lint || exit 1; \
		done; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@for dir in $(COMPONENTS); do \
		$(MAKE) -C $$dir clean || exit 1; \
	done
	@find . -type f -name "*.test" -delete
	@find . -type f -name "coverage.out" -delete
	@find . -type f -name "coverage.html" -delete
	@find . -type d -name "vendor" -exec rm -rf {} + 2>/dev/null || true
	@rm -rf bin/
