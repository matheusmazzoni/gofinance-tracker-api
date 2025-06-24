# Build parameters
CMD_PATH=./cmd/api
BINARY_NAME=finance-tracker
DOCKER_REGISTRY=ghcr.io
DOCKER_IMAGE_NAME=$(DOCKER_REGISTRY)/$(shell git config --get remote.origin.url | sed 's/.*github.com[:\/]\([^\/]*\/[^\.]*\).*/\1/')/$(BINARY_NAME)
DOCKER_TAG?=$(shell git rev-parse --short HEAD)
PLATFORMS?=linux/amd64,linux/arm64

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOSWAG=swag

# Test output
TEST_OUTPUT?=test-report.xml

.PHONY: all build run clean test coverage download tidy lint swag-docs docker-* help

all: help

build:
	@echo "Building application..."
	@$(GOBUILD) -ldflags="-s -w" -o bin/$(BINARY_NAME) $(CMD_PATH)

run: build
	@echo "Running application..."
	@./bin/$(BINARY_NAME)

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f $(TEST_OUTPUT)
	@rm -f coverage.out

download:
	@echo "Downloading go modules..."
	@$(GOMOD) download

tidy:
	@echo "Tidying go modules..."
	@$(GOMOD) tidy

test:
	@echo "Running tests..."
	@$(GOTEST) -v -race -cover ./...

test-ci:
	@echo "Running tests with JUnit output..."
	@$(GOTEST) -v -race -cover ./... 2>&1 | go-junit-report -set-exit-code > $(TEST_OUTPUT)

coverage:
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@$(GOCMD) tool cover -html=coverage.out

lint:
	@echo "Running linters..."
	@golangci-lint run ./...

start: 
	@echo "Starting environment..."
	@docker compose build
	@docker compose up -d

swag-docs:
	@echo "Generating Swagger documentation..."
	@$(GOSWAG) fmt
	@$(GOSWAG) init -g $(CMD_PATH)/main.go -o ./api

# Docker targets
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .

docker-run: docker-build
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --rm --name $(BINARY_NAME)-container $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)

docker-push:
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)

docker-build-multiarch:
	@echo "Building multi-architecture Docker images..."
	@docker buildx build --platform $(PLATFORMS) -t $(DOCKER_IMAGE_NAME):$(DOCKER_TAG) --push .

docker-scan:
	@echo "Scanning Docker image for vulnerabilities..."
	@trivy image $(DOCKER_IMAGE_NAME):$(DOCKER_TAG)

help:
	@echo "Available commands:"
	@echo "  build                  - Builds the application"
	@echo "  start                  - Runs the application and db locally"
	@echo "  run                    - Runs the application locally"
	@echo "  clean                  - Removes build artifacts"
	@echo "  download               - Downloads go modules"
	@echo "  tidy                   - Tidies up go module dependencies"
	@echo "  test                   - Runs all tests"
	@echo "  test-ci                - Runs tests with JUnit output for CI"
	@echo "  coverage               - Runs tests with coverage report"
	@echo "  lint                   - Runs linters"
	@echo "  swag-docs              - Generates Swagger documentation"
	@echo "  docker-build           - Builds the Docker image"
	@echo "  docker-run             - Runs a container from the built image"
	@echo "  docker-push            - Pushes the Docker image to registry"
	@echo "  docker-build-multiarch - Builds and pushes multi-arch images"
	@echo "  docker-scan            - Scans Docker image for vulnerabilities"