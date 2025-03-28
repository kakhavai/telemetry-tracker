IMAGE_NAME ?= kakhavain/telemetry-tracker

.PHONY: all build run docker-build docker-run docker-push clean

# Default target: build the Go binary
all: build

# Build the Go binary
build:
	@echo "Building the telemetry-tracker binary..."
	go build -o telemetry-tracker main.go

# Run the binary locally
run: build
	@echo "Running telemetry-tracker..."
	./telemetry-tracker

# Build the Docker image
docker-build:
	@echo "Building Docker image: $(IMAGE_NAME)..."
	docker build -t $(IMAGE_NAME) .

# Run the Docker container locally, mapping port 8080
docker-run: docker-build
	@echo "Running Docker container from image: $(IMAGE_NAME)..."
	docker run -p 8080:8080 $(IMAGE_NAME)

# Push the Docker image to your registry (DockerHub/ECR)
docker-push:
	@echo "Pushing Docker image: $(IMAGE_NAME)..."
	docker push $(IMAGE_NAME)

# Clean up the built binary
clean:
	@echo "Cleaning up..."
	rm -f telemetry-tracker
