# PulseDB Makefile

.PHONY: build test clean run demo docker help

# Default target
help:
	@echo "PulseDB - Redis-like in-memory database with MVCC"
	@echo ""
	@echo "Available targets:"
	@echo "  build    - Build the PulseDB binary"
	@echo "  test     - Run all tests"
	@echo "  test-v   - Run tests with verbose output"
	@echo "  bench    - Run benchmarks"
	@echo "  run      - Run PulseDB server"
	@echo "  demo     - Run feature demonstration"
	@echo "  clean    - Clean build artifacts"
	@echo "  deps     - Download dependencies"
	@echo "  lint     - Run linter"
	@echo "  format   - Format code"
	@echo "  docker   - Build Docker image"
	@echo "  help     - Show this help"

# Build the binary
build:
	@echo "Building PulseDB..."
	go build -o pulsedb cmd/pulsedb/main.go
	@echo "✅ Build complete: ./pulsedb"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output  
test-v:
	@echo "Running tests (verbose)..."
	go test -v ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. ./...

# Run with coverage
test-cover:
	@echo "Running tests with coverage..."
	go test -v -cover ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f pulsedb
	go clean

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Format code
format:
	@echo "Formatting code..."
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run the server
run: build
	@echo "Starting PulseDB..."
	./pulsedb

# Run the demo
demo: build
	@echo "Running PulseDB demo..."
	./demo.sh

# Install tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Build examples
examples:
	@echo "Building examples..."
	@if [ -f examples/client.go ]; then \
		go build -o examples/client examples/client.go; \
		echo "✅ Client example built: examples/client"; \
	fi

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t pulsedb:latest .

# Development workflow
dev: deps format lint test build
	@echo "✅ Development workflow complete"

# CI workflow  
ci: deps test bench
	@echo "✅ CI workflow complete"

# Full build and test
all: clean deps format lint test bench build examples
	@echo "✅ Full build complete"
