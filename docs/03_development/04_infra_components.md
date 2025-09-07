# Using `im-infra` Components

The `im-infra` directory contains shared libraries used across all microservices. This guide explains how to use the key components.

## 1. `clog` - Structured Logging

The `clog` library provides a standardized, structured logging interface.

-   **Initialization**: A logger is typically initialized in `main.go` and passed down to other components.
-   **Usage**:
    ```go
    import "github.com/ceyewan/gochat/im-infra/clog"

    // Get a logger for a specific module
    logger := clog.Module("user-service")

    // Log messages with structured context
    logger.Info("User logged in",
        clog.String("username", "test"),
        clog.Uint64("user_id", 123),
    )

    logger.Error("Failed to connect to database", clog.Err(err))
    ```
-   **Configuration**: The logger is configured via the `clog.json` file, which is loaded by the `coord` service. See the [Configuration Management](./../../04_deployment/02_configuration.md) guide for details.

## 2. `coord` - Distributed Coordination

The `coord` library provides an interface for distributed coordination services, including service discovery, configuration management, and distributed locks, using `etcd` as the backend.

-   **Initialization**: A `coord.Provider` is created in `main.go` and is used to access the different coordination services.
    ```go
    import "github.com/ceyewan/gochat/im-infra/coord"

    // cfg is loaded from the config file
    coordinator, err := coord.New(context.Background(), cfg)
    if err != nil {
        // handle error
    }
    defer coordinator.Close()
    ```

### Service Discovery & gRPC

-   The `coord` library integrates with gRPC to provide client-side load balancing.
-   **Getting a gRPC client connection**:
    ```go
    // Get a connection to the "user-service"
    conn, err := coordinator.Registry().GetConnection(ctx, "user-service")
    if err != nil {
        // handle error
    }
    defer conn.Close()

    // Create a gRPC client
    userClient := userpb.NewUserServiceClient(conn)
    ```

### Configuration Management

-   Services use the `coord` provider to fetch their configuration from `etcd`.
-   **Fetching a configuration**:
    ```go
    var dbConfig myapp.DatabaseConfig
    err := coordinator.Config().Get(ctx, "/config/dev/im-repo/db", &dbConfig)
    if err != nil {
        // handle error
    }
    ```

### Distributed Locking

-   The `coord` library provides a simple interface for acquiring and releasing distributed locks.
-   **Acquiring a lock**:
    ```go
    // Acquire a lock with a 30-second TTL
    lock, err := coordinator.Lock().Acquire(ctx, "my-resource-key", 30*time.Second)
    if err != nil {
        // handle error (e.g., lock already held)
    }
    defer lock.Unlock(ctx)

    // ... perform critical section work ...
    ```

For more detailed information on the design and capabilities of the `coord` module, refer to its [design document](../../../im-infra/coord/DESIGN.md) and [README](../../../im-infra/coord/README.md).
