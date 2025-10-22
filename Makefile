# Makefile for go-openzl

.PHONY: all build test bench clean build-openzl help fmt lint ci install-tools

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Directories
VENDOR_DIR=vendor
OPENZL_DIR=$(VENDOR_DIR)/openzl
OPENZL_LIB=$(OPENZL_DIR)/lib/libopenzl.a

# Default target
all: test build

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the Go package
build:
	$(GOBUILD) -v ./...

## test: Run tests with race detector
test:
	$(GOTEST) -v -race ./...

## test-short: Run tests without race detector (faster)
test-short:
	$(GOTEST) -v -short ./...

## bench: Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem -run=^$$ ./...

## coverage: Generate test coverage report
coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## fmt: Format Go code
fmt:
	$(GOFMT) -s -w .
	$(GOMOD) tidy

## lint: Run linters
lint:
	$(GOLINT) run

## ci: Run CI checks (fmt, lint, test)
ci: fmt lint test

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest

## build-openzl: Build the OpenZL C library
build-openzl: check-openzl
	@echo "Building OpenZL C library..."
	cd $(OPENZL_DIR) && $(MAKE) lib BUILD_TYPE=OPT
	@mkdir -p $(OPENZL_DIR)/lib
	@cp $(OPENZL_DIR)/libopenzl.a $(OPENZL_DIR)/lib/
	@echo "OpenZL library built successfully at $(OPENZL_LIB)"

## check-openzl: Check if OpenZL source exists
check-openzl:
	@if [ ! -d "$(OPENZL_DIR)" ]; then \
		echo "Error: OpenZL source not found at $(OPENZL_DIR)"; \
		echo ""; \
		echo "Please set up OpenZL source:"; \
		echo "  Option 1: git submodule add https://github.com/facebook/openzl.git vendor/openzl"; \
		echo "  Option 2: git clone https://github.com/facebook/openzl.git vendor/openzl"; \
		echo ""; \
		echo "See vendor/README.md for details"; \
		exit 1; \
	fi

## clean: Clean build artifacts
clean:
	$(GOCMD) clean
	rm -f coverage.out coverage.html
	@if [ -d "$(OPENZL_DIR)" ]; then \
		cd $(OPENZL_DIR) && $(MAKE) clean; \
	fi

## clean-all: Clean everything including OpenZL build
clean-all: clean
	@if [ -d "$(OPENZL_DIR)" ]; then \
		rm -rf $(OPENZL_DIR)/lib; \
	fi

## examples: Build example programs
examples:
	@echo "TODO: Build examples when implemented"

## verify: Verify module dependencies
verify:
	$(GOMOD) verify

## tidy: Tidy module dependencies
tidy:
	$(GOMOD) tidy

.DEFAULT_GOAL := help
