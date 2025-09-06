# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoChat is a distributed instant messaging system built with Go using a microservices architecture. The system consists of four core services: im-gateway (API/WebSocket), im-logic (business logic), im-repo (data layer), and im-task (async processing).

## Essential Commands

### Development Setup
```bash
# Start development environment (MySQL, Redis, Kafka, etcd)
make dev

# Install development tools (golangci-lint, protoc plugins)
make install-tools

# Download dependencies
make deps
```

### Daily Development
```bash
# Format code
make fmt

# Lint code (installs golangci-lint if needed)
make lint

# Generate protobuf code after .proto changes
make proto

# Run all tests with race detection and coverage
make test

# Test specific service
make test-service SERVICE=im-gateway
```

### Building
```bash
# Build all services
make build

# Build specific service
make build-service SERVICE=im-logic

# Build Docker images
make docker-build
```

### Cleanup
```bash
# Stop development environment
make dev-down

# Clean build artifacts
make clean
```

## Architecture Overview

### Service Communication Flow
```
Client → WebSocket → im-gateway → Kafka (upstream) → im-logic → gRPC → im-repo
                                           ↓
im-logic → Kafka (downstream) → im-gateway → WebSocket → Client
                                           ↓
im-logic → Kafka (task) → im-task (async processing)
```

### Service Responsibilities
- **im-gateway**: Client WebSocket/HTTP connections, message routing (port 8080)
- **im-logic**: Business logic, authentication, message processing (gRPC port 9001)
- **im-repo**: Data persistence, user/message storage (gRPC port 9002)
- **im-task**: Background tasks, large group fanout (Kafka consumer)

### Key Infrastructure
- **MySQL**: Primary data storage (users, messages, conversations, groups)
- **Redis**: Caching and session management
- **Kafka**: Message queue (topics: im-upstream-topic, im-downstream-topic-{gateway_id}, im-task-topic)
- **etcd**: Service discovery and configuration center

### Communication Patterns
- **gRPC**: Synchronous service-to-service calls (gateway→logic, logic→repo)
- **Kafka**: Asynchronous message passing between services
- **WebSocket**: Real-time client connections

### Message Flow
1. Client sends message via WebSocket to im-gateway
2. im-gateway publishes to Kafka upstream topic
3. im-logic consumes, validates, processes business logic
4. im-logic calls im-repo via gRPC to persist data
5. im-logic publishes to Kafka downstream topic for delivery
6. im-gateway delivers to target clients via WebSocket

## Development Guidelines

### Working with Protobuf
- All service APIs are defined in `/api/proto/`
- Run `make proto` after modifying .proto files
- Generated code goes to `/api/gen/`

### Testing
- Use Go's built-in testing with testify assertions
- Race detection is enabled by default (`-race` flag)
- Coverage reports generated in `coverage.html`
- Test service-specific code with `make test-service SERVICE=<service>`

### Configuration
- YAML-based configs with environment variable overrides
- Service configs in each service's `/config/` directory
- Runtime config updates via etcd

### Database
- MySQL schema initialization in `/scripts/mysql/init.sql`
- GORM for ORM with support for MySQL, PostgreSQL, SQLite
- Migration scripts should be added to `/scripts/mysql/`

### Error Handling
- Structured error responses with error codes
- Circuit breaker patterns for service protection
- Graceful shutdown handling
- Health check endpoints for all services

### Logging
- Structured JSON logging with contextual information
- Log levels: debug, info, warn, error
- Use contextual logging with request IDs for tracing

## Key Dependencies
- **Web Framework**: Gin for HTTP, gorilla/websocket for WebSocket
- **gRPC**: Service-to-service communication
- **Kafka**: franz-go client for message queuing
- **Database**: GORM with MySQL/PostgreSQL/SQLite support
- **Redis**: go-redis for caching
- **Authentication**: JWT with golang.org/x/crypto
- **Observability**: OpenTelemetry for tracing, Prometheus for metrics