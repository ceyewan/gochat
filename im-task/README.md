# im-task Service

im-task is a background task processing service for the GoChat distributed instant messaging system. It handles large group message fanout, message persistence, and offline push notifications.

## Features

- **Large Group Message Fanout**: Efficiently distributes messages to large groups using Kafka
- **Message Persistence**: Acts as a separate consumer group for persisting messages to the database
- **Gateway Instance Discovery**: Queries user gateway instances for message routing
- **Offline Push Notifications**: Handles push notifications for offline users
- **Retry Mechanism**: Implements retry logic for failed operations
- **Monitoring**: Integrated with Prometheus, Grafana, and Jaeger

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   im-logic      │    │   im-task       │    │   im-repo       │
│                 │    │                 │    │                 │
│  ┌───────────┐  │    │  ┌───────────┐  │    │  ┌───────────┐  │
│  │  Kafka    │  │    │  │  Kafka    │  │    │  │  gRPC     │  │
│  │ Producer  │  │────│  │ Consumer  │  │────│  │ Server    │  │
│  └───────────┘  │    │  └───────────┘  │    │  └───────────┘  │
│                 │    │                 │    │                 │
│  ┌───────────┐  │    │  ┌───────────┐  │    │  ┌───────────┐  │
│  │  gRPC     │  │    │  │  gRPC     │  │    │  │  MySQL    │  │
│  │ Client    │  │────│  │ Client    │  │────│  │ Database  │  │
│  └───────────┘  │    │  └───────────┘  │    │  └───────────┘  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Configuration

The service configuration is located in `configs/config.yaml`. Key configuration sections:

### Kafka Configuration
- `task_topic`: Topic for task messages
- `persistence_topic`: Topic for persistence messages
- `consumer_group`: Consumer group for task processing
- `persistence_group`: Consumer group for message persistence

### Task Configuration
- `large_group_threshold`: Minimum group size for fanout processing
- `fanout_batch_size`: Batch size for user fanout operations
- `push_notification_enabled`: Enable offline push notifications
- `message_retry_attempts`: Number of retry attempts for failed operations

## Quick Start

### Prerequisites
- Go 1.21+
- Docker and Docker Compose
- Kafka, Redis, MySQL

### Development Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd gochat/im-task
   ```

2. **Install dependencies**
   ```bash
   make deps
   ```

3. **Start development environment**
   ```bash
   make dev
   ```

4. **Build and run the service**
   ```bash
   make build
   ./bin/im-task
   ```

### Using Docker

1. **Build Docker image**
   ```bash
   make docker-build
   ```

2. **Start all services**
   ```bash
   make docker-up
   ```

3. **Stop services**
   ```bash
   make docker-down
   ```

## API Documentation

### Task Message Types

The service handles different types of task messages:

1. **Large Group Fanout** (`TASK_MESSAGE_TYPE_LARGE_GROUP_FANOUT`)
   - Processes messages for large groups
   - Queries user gateway instances
   - Fans out messages to appropriate gateways

2. **Offline Push** (`TASK_MESSAGE_TYPE_OFFLINE_PUSH`)
   - Sends push notifications to offline users
   - Handles different message types (text, image, file, voice)

3. **Message Retry** (`TASK_MESSAGE_TYPE_MESSAGE_RETRY`)
   - Retries failed message delivery
   - Implements exponential backoff

### Message Flow

1. **Task Message Processing**
   ```
   im-task-topic → im-task → im-repo (query users) → downstream-topic-{gateway_id}
   ```

2. **Message Persistence**
   ```
   im-persistence-topic → im-task → im-repo (persist message)
   ```

3. **Push Notifications**
   ```
   im-task → im-push-topic → Push Service
   ```

## Monitoring

### Metrics
The service exposes metrics on port 9093 at `/metrics`:

- Kafka consumer lag
- Message processing rates
- Error rates
- gRPC client health

### Health Check
Health endpoint available at `/health` on port 9093.

### Observability
- **Prometheus**: Metrics collection and alerting
- **Grafana**: Dashboard visualization
- **Jaeger**: Distributed tracing

## Development

### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage
make test-service
```

### Code Quality
```bash
# Format code
make fmt

# Lint code
make lint
```

### Hot Reload
```bash
# Run with air for hot reload
make run-dev
```

## Troubleshooting

### Common Issues

1. **Kafka Connection Issues**
   - Check if Kafka is running: `docker-compose ps kafka`
   - Verify broker addresses in config
   - Check network connectivity

2. **gRPC Connection Issues**
   - Verify im-repo service is running
   - Check gRPC client configuration
   - Verify network connectivity

3. **Message Processing Errors**
   - Check Kafka topic configurations
   - Verify consumer group settings
   - Check message format and validation

### Logs
Check service logs for detailed error information:
```bash
docker-compose logs im-task
```

## License

This project is licensed under the MIT License.