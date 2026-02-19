.PHONY: all build test test-verbose test-coverage coverage-check clean fmt vet lint install run docker

GO ?= go
BINARY := aima
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := bin
COVERAGE_OUT := coverage.out
COVERAGE_HTML := coverage.html
COVERAGE_THRESHOLD := 60

all: fmt vet build

build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd/aima

test:
	$(GO) test ./... -cover -race

test-verbose:
	$(GO) test ./... -cover -race -v

test-coverage:
	$(GO) test ./... -coverprofile=$(COVERAGE_OUT) -covermode=atomic -race
	$(GO) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

coverage-check: test-coverage
	@total=$$($(GO) tool cover -func=$(COVERAGE_OUT) | grep '^total:' | awk '{print $$3}' | tr -d '%'); \
	echo "Total coverage: $${total}%"; \
	if [ $$(echo "$${total} < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "ERROR: coverage $${total}% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi; \
	echo "OK: coverage $${total}% meets threshold $(COVERAGE_THRESHOLD)%"

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	golangci-lint run

install:
	$(GO) install ./cmd/aima

run:
	$(BUILD_DIR)/$(BINARY) start

clean:
	rm -rf $(BUILD_DIR) $(COVERAGE_OUT) $(COVERAGE_HTML)

docker:
	docker build -t aima:$(VERSION) .

help:
	@echo "Available targets:"
	@echo "  build            - Build the binary"
	@echo "  test             - Run tests with race detection"
	@echo "  test-verbose     - Run tests with verbose output"
	@echo "  test-coverage    - Run tests and generate HTML coverage report"
	@echo "  coverage-check   - Run tests and fail if coverage is below $(COVERAGE_THRESHOLD)%"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  lint             - Run golangci-lint"
	@echo "  install          - Install binary"
	@echo "  clean            - Clean build artifacts and coverage files"
	@echo "  docker           - Build Docker image"
