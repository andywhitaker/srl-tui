# Makefile for srl-tui cross-compilation and local testing

BINARY_NAME := srl-tui
BUILD_DIR := dist

.PHONY: all build test clean build-all build-linux-amd64 build-linux-arm64

all: test build

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY_NAME) .

test:
	go test -v ./...
	go vet ./...

clean:
	rm -rf $(BUILD_DIR) $(BINARY_NAME)

build-all: build-linux-amd64 build-linux-arm64

build-linux-amd64:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .

build-linux-arm64:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
