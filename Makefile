.PHONY: all build-server build-sidecar clean test lint docker-server docker-sidecar

BINARY_DIR := bin
SERVER_BIN := $(BINARY_DIR)/server
SIDECAR_BIN := $(BINARY_DIR)/sidecar

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

all: build-server build-sidecar

build-server:
	@echo "==> Building server..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(SERVER_BIN) ./cmd/server

build-sidecar:
	@echo "==> Building sidecar..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(SIDECAR_BIN) ./cmd/sidecar

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BINARY_DIR)

docker-server:
	docker build -t sidecar-server:$(VERSION) -f deploy/Dockerfile.server .

docker-sidecar:
	docker build -t sidecar-client:$(VERSION) -f deploy/Dockerfile.sidecar .

# Cross-compile for multiple platforms
release:
	@echo "==> Cross-compiling..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/server-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/sidecar-linux-amd64 ./cmd/sidecar
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_DIR)/server-linux-arm64 ./cmd/server
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_DIR)/sidecar-linux-arm64 ./cmd/sidecar
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/server-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/sidecar-darwin-amd64 ./cmd/sidecar
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_DIR)/server-darwin-arm64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_DIR)/sidecar-darwin-arm64 ./cmd/sidecar
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/server-windows-amd64.exe ./cmd/server
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/sidecar-windows-amd64.exe ./cmd/sidecar
