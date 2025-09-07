.PHONY: build build-service test test-service clean docker-build docker-up docker-down deps fmt lint

# Build targets
build:
	go build -o bin/im-task ./cmd/server

build-service:
	go build -o bin/im-task ./cmd/server

# Test targets
test:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

test-service:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean target
clean:
	rm -rf bin/ coverage.out coverage.html

# Docker targets
docker-build:
	docker build -t im-task:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Development targets
deps:
	go mod download
	go mod tidy

fmt:
	go fmt ./...

lint:
	golangci-lint run

# Development environment
dev:
	docker-compose up -d kafka redis mysql im-repo

dev-down:
	docker-compose down

# Run service
run:
	go run ./cmd/server

# Run with hot reload
run-dev:
	air -c .air.toml
