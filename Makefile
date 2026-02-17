.PHONY: all build test clean fmt vet lint install run docker

GO ?= go
BINARY := aima
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := bin

all: fmt vet build

build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd/aima

test:
	$(GO) test ./... -cover

test-verbose:
	$(GO) test ./... -cover -v

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
	rm -rf $(BUILD_DIR)

docker:
	docker build -t aima:$(VERSION) .

help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run golangci-lint"
	@echo "  install      - Install binary"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker       - Build Docker image"
