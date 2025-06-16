BINARY_NAME=gofinance-tracker-api-app
CMD_PATH=./cmd/api
DOCKER_IMAGE_NAME=gofinance-tracker-api

.PHONY: all build run tidy test docker-build docker-run help

all: help

build:
	@echo "Building binary..."
	@go build -o $(BINARY_NAME) $(CMD_PATH)

run:
	@echo "Running application..."
	@go run $(CMD_PATH)

download:
	@echo "Download go modules..."
	@go mod download

tidy:
	@echo "Tidying go modules..."
	@go mod tidy

test:
	@echo "Running tests..."
	@go test -v -race -cover ./...

swag-docs:
	@swag fmt
	@swag init -g $(CMD_PATH)/main.go 

docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE_NAME) .

docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --rm --name $(DOCKER_IMAGE_NAME)-container $(DOCKER_IMAGE_NAME)

help:
	@echo "Available commands:"
	@echo "  build         - Compiles the application binary"
	@echo "  run           - Runs the application locally for development"
	@echo "  tidy          - Tidies up go module dependencies"
	@echo "  test          - Runs all tests"
	@echo "  docker-build  - Builds the Docker image"
	@echo "  docker-run    - Runs a container from the built image"