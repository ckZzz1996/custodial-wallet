.PHONY: all build run test clean docker-build docker-run migrate proto

# Variables
APP_NAME := custodial-wallet
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build

# Directories
CMD_API := ./cmd/api
CMD_WORKER := ./cmd/worker
BIN_DIR := ./bin
PROTO_DIR := ./api/proto

all: build

# Proto generation
proto:
	@echo "Generating proto files..."
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(PROTO_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/wallet/v1/*.proto
	@echo "Proto files generated!"

proto-install:
	@echo "Installing protoc plugins..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Build
build: build-api build-worker

build-api:
	@echo "Building API..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/api $(CMD_API)

build-worker:
	@echo "Building Worker..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/worker $(CMD_WORKER)

# Run
run-api:
	@echo "Running API..."
	$(GO) run $(CMD_API)/main.go

run-worker:
	@echo "Running Worker..."
	$(GO) run $(CMD_WORKER)/main.go

# Test
test:
	@echo "Running tests..."
	$(GOTEST) -v -cover ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html

# Dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Docker
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 $(APP_NAME):$(VERSION)

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down

# Database
migrate:
	@echo "Running migrations..."
	$(GO) run $(CMD_API)/main.go migrate

# Generate
generate:
	@echo "Generating code..."
	$(GO) generate ./...

# Swagger
swagger:
	@echo "Generating Swagger docs..."
	swag init -g $(CMD_API)/main.go -o ./docs/swagger

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build all binaries"
	@echo "  build-api     - Build API server"
	@echo "  build-worker  - Build worker"
	@echo "  run-api       - Run API server"
	@echo "  run-worker    - Run worker"
	@echo "  test          - Run tests"
	@echo "  lint          - Run linter"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  help          - Show this help"

