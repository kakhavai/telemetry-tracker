# Makefile

.PHONY: all build run clean test lint tidy docker-build \
        compose-up compose-down compose-logs compose-ps help

# Variables
APP_NAME := telemetry-tracker
BINARY_NAME := server
DOCKER_IMAGE_NAME := $(APP_NAME)
DOCKER_TAG := latest
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")

# Default target
all: build

# Build the Go binary for the current OS/Arch
build: tidy
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) ./cmd/server/main.go

# Run the Go application locally (requires DB running and env vars EXPORTED)
run: build
	@echo "Running $(BINARY_NAME) locally..."
	@echo "Ensure DB is running and ENV VARS are EXPORTED (e.g., DB_HOST=localhost)"
	@./$(BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Tidy Go modules
tidy:
	@echo "Tidying modules..."
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf ./vendor/

# --- Docker ---

# Build the Docker image (used implicitly by compose-up if needed)
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)..."
	@docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .

# --- Docker Compose ---

# Start services defined in docker-compose.yml in detached mode. Builds images if necessary.
compose-up:
	@echo "Starting Docker Compose services (app & postgres)..."
	@if [ ! -f .env ]; then \
		echo "ERROR: .env file not found."; \
		echo "Please copy .env.example to .env and fill in your database credentials (set DB_HOST=postgres)."; \
		exit 1; \
	fi
	@docker-compose up -d --build

# Stop and remove containers, networks defined in docker-compose.yml.
compose-down:
	@echo "Stopping Docker Compose services..."
	@docker-compose down

# Follow logs from the 'app' service.
compose-logs:
	@echo "Following logs for the 'app' service..."
	@docker-compose logs -f app

# Show status of containers managed by docker-compose.
compose-ps:
	@echo "Showing status of Docker Compose services..."
	@docker-compose ps

# Show help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Development Targets:'
	@echo '  build          Build the Go binary locally'
	@echo '  run            Build and run locally (requires env vars exported, DB running separately)'
	@echo '  test           Run Go tests'
	@echo '  lint           Run Go linter (golangci-lint)'
	@echo '  tidy           Run go mod tidy'
	@echo '  clean          Remove build artifacts'
	@echo ''
	@echo 'Docker Compose Targets (Recommended for Development):'
	@echo '  compose-up     Build images (if needed) and start app & postgres containers using docker-compose. Requires .env file.'
	@echo '  compose-down   Stop and remove containers managed by docker-compose.'
	@echo '  compose-logs   Follow logs from the running app container.'
	@echo '  compose-ps     Show status of docker-compose managed containers.'
	@echo ''
	@echo 'Docker Targets:'
	@echo '  docker-build   Build the Docker image manually'
	@echo ''