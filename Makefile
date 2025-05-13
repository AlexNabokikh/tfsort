# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOINSTALL=$(GOCMD) install
BINARY_NAME=tfsort
COVERAGE_FILE=c.out

LINTCMD=golangci-lint run

# Git versioning
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
GIT_COMMIT_HASH := $(shell git rev-parse HEAD 2>/dev/null)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null)

# Provide defaults if git commands fail or not in a git repo
VERSION ?= $(if $(GIT_TAG),$(GIT_TAG),dev)
COMMIT ?= $(if $(GIT_COMMIT_HASH),$(GIT_COMMIT_HASH),unknown)
DATE ?= $(if $(BUILD_DATE),$(BUILD_DATE),unknown)

# Build flags for injecting version information into main.go
LDFLAGS = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.DEFAULT_GOAL := help

.PHONY: all build test coverage lint clean install run help setup-lint

all: build

build:
	@echo "Building $(BINARY_NAME) version $(VERSION) (commit: $(COMMIT), built: $(DATE))..."
	$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./main.go

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

coverage:
	@echo "Running tests and generating coverage report ($(COVERAGE_FILE))..."
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) ./...
	@sed -i "s%github.com/AlexNabokikh/%%" $(COVERAGE_FILE)

lint:
	@echo "Linting code using golangci-lint..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo >&2 "golangci-lint not found. Please install it first."; \
		echo >&2 "See: https://golangci-lint.run/usage/install/"; \
		echo >&2 "Alternatively, run 'make setup-lint' to try to install it."; \
		exit 1; \
	}
	$(LINTCMD)

setup-lint:
	@echo "Attempting to install golangci-lint..."
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

clean:
	@echo "Cleaning up build artifacts and coverage files..."
	$(GOCLEAN) -cache
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE)
	@echo "Cleanup complete."

install:
	@echo "Installing $(BINARY_NAME) with version information..."
	$(GOINSTALL) -ldflags="$(LDFLAGS)" ./...
	@echo "$(BINARY_NAME) installed successfully."

run: build
	@echo "Running $(BINARY_NAME) $(ARGS)..."
	./$(BINARY_NAME) $(ARGS)

help:
	@echo "Makefile for the $(BINARY_NAME) project"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Common targets:"
	@echo "  all         Build the application binary (same as 'build')."
	@echo "  build       Build the application binary with embedded version information."
	@echo "              Override version details: make build VERSION=1.0.1 COMMIT=mycommit DATE=mydate"
	@echo "  test        Run all Go tests."
	@echo "  coverage    Run tests and generate a code coverage report (outputs to $(COVERAGE_FILE))."
	@echo "  lint        Lint the Go source code using golangci-lint."
	@echo "  clean       Remove build artifacts (the binary, coverage files) and clear Go build cache."
	@echo "  install     Install the application binary to your Go bin path (e.g., $$GOPATH/bin)."
	@echo "  run         Build and then run the application. Pass arguments via ARGS variable."
	@echo "              Example: make run ARGS=\"input.tf -o output.tf\""
	@echo "  help        Show this help message (default target)."
	@echo ""
	@echo "Other targets:"
	@echo "  setup-lint  Attempt to install golangci-lint using 'go install'."
	@echo ""
	@echo "Build Variables:"
	@echo "  VERSION     Set the version string (default: latest git tag or 'dev')."
	@echo "  COMMIT      Set the commit hash (default: current git commit hash or 'unknown')."
	@echo "  DATE        Set the build date (default: current UTC date/time or 'unknown')."

