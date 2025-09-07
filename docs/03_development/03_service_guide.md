# Microservice Development Guide

This guide provides instructions and best practices for developing, running, and testing individual microservices within the GoChat project.

## 1. Service Structure

Each microservice (e.g., `im-logic`, `im-repo`) follows a consistent directory structure:

-   **/cmd**: Contains the `main.go` file, which is the entry point for the service. Its primary responsibility is to initialize and start the service.
-   **/internal**: Contains all the private code for the service.
    -   **/config**: Handles loading and managing configuration.
    -   **/model**: Defines the data structures (structs) used within the service, particularly for database interaction.
    -   **/repository**: The data access layer. It contains the logic for interacting with the database and cache.
    -   **/service**: The business logic layer. It implements the gRPC service interfaces defined in the `.proto` files.
-   **/api**: This directory is at the project root and contains all `.proto` definitions and generated code.

## 2. Running a Service Locally

Each microservice can be run independently for development and testing.

1.  **Prerequisites**: Ensure the necessary infrastructure (database, cache, etc.) is running. You can start it with the script in the `deployment` directory:
    ```bash
    ./deployment/scripts/start-infra.sh
    ```
2.  **Run the Service**: Navigate to the service's directory and use the `go run` command:
    ```bash
    cd im-repo
    go run ./cmd/server/main.go
    ```

## 3. Adding a New gRPC Method

1.  **Define in `.proto`**: Add the new RPC method to the appropriate service definition in the relevant `.proto` file in `api/proto/`.
2.  **Generate Code**: Run `buf generate` from the `api/` directory to generate the updated Go interfaces and client/server code.
    ```bash
    cd api
    buf generate
    ```
3.  **Implement in Repository (if needed)**: If the new method requires data access, add the corresponding function to the appropriate repository file in `im-repo/internal/repository/`.
4.  **Implement in Service**: Implement the new gRPC method in the corresponding service file (e.g., `im-logic/internal/service/user_service.go`). This is where the business logic resides.
5.  **Write Tests**: Add unit tests for the new repository and service functions.

## 4. Testing

-   **Unit Tests**: Each new function in the `service` and `repository` layers should have corresponding unit tests.
    -   Use mocks for dependencies (e.g., mock the repository when testing the service).
    -   Test files should be named `_test.go`.
-   **Integration Tests**: For more complex features, integration tests can be added to test the interaction between services. These tests typically run against a real database and other infrastructure components spun up via Docker.
-   **Running Tests**:
    ```bash
    # Run tests for a specific package
    go test ./im-repo/internal/service

    # Run all tests in the project
    go test ./...
    ```

## 5. Dependency Management

-   Dependencies are managed using Go Modules.
-   To add a new dependency, use `go get <package-name>`.
-   After adding or updating dependencies, run `go mod tidy` to clean up the `go.mod` and `go.sum` files.
