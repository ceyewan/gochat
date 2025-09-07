# Configuration Management

This document explains how to manage configuration for the GoChat microservices.

## 1. Overview

GoChat uses a centralized configuration management system powered by `etcd`. Configuration files are written in JSON and stored in the `/config/dev` directory. A command-line tool, `config-cli`, is provided to sync these local files to the `etcd` server.

## 2. Configuration Structure

-   **Local Files**: All configuration files are located in `config/dev/`. Each service has its own subdirectory, and each component within that service has its own JSON file.
    -   Example: `config/dev/im-repo/db.json`
-   **etcd Path**: The configuration path in `etcd` follows a strict schema:
    -   `/config/{environment}/{service}/{component}`
    -   Example: `/config/dev/im-repo/db`

## 3. `config-cli` Tool

The `config-cli` tool is used to synchronize the local JSON configuration files with the `etcd` server.

-   **Location**: `config/config-cli/`
-   **Usage**:
    ```bash
    # Navigate to the tool's directory
    cd config/config-cli

    # Sync all configurations for the 'dev' environment
    ./config-cli sync dev

    # Sync all configurations for a specific service
    ./config-cli sync dev im-repo

    # Sync a single component's configuration
    ./config-cli sync dev im-repo db
    ```

## 4. Configuration Loading in Services

Microservices load their configuration in a two-stage process to ensure resilience.

1.  **Bootstrap Phase**: On startup, the service loads a minimal, hard-coded default configuration. This is just enough to initialize the logging and `coord` (etcd client) components.
2.  **Full Configuration Load**: The service then uses the `coord` component to connect to `etcd` and fetch its full configuration.
    -   If the connection to `etcd` fails, the service will continue to run with the default configuration, ensuring it can still start up even if the configuration service is temporarily unavailable.
    -   The `coord` component also watches for changes in `etcd`, allowing for dynamic, hot-reloading of configuration without restarting the service.

## 5. Configuration Schema Examples

### `db.json` (Database)

```json
{
  "dsn": "user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
  "driver": "mysql",
  "maxOpenConns": 25,
  "maxIdleConns": 10
}
```

### `cache.json` (Redis Cache)

```json
{
  "addr": "redis:6379",
  "password": "",
  "db": 0
}
```

### `coord.json` (etcd)

```json
{
  "endpoints": ["etcd1:2379", "etcd2:2379", "etcd3:2379"],
  "timeout": "5s"
}
```

### `clog.json` (Logging)

```json
{
  "level": "info",
  "format": "json",
  "output": "file",
  "filename": "/app/logs/app.log"
}
