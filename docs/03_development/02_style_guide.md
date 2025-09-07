# GoChat Code Style and Conventions

This document defines the coding style, formatting, and commenting standards for the GoChat project. Adhering to these conventions is crucial for maintaining code quality, readability, and consistency across the codebase.

## 1. Go Language Style

-   **Formatting**: All Go code **must** be formatted with `gofmt`. Most IDEs can be configured to do this automatically on save.
-   **Linting**: We use `golangci-lint` for static analysis. All code must pass the linter checks before being merged. The configuration for the linter can be found in the `.golangci.yml` file at the project root.
-   **Error Handling**:
    -   Errors should be handled explicitly. Do not ignore errors using the blank identifier (`_`).
    -   Error messages should be in lowercase and not capitalized.
    -   Use `fmt.Errorf` with the `%w` verb to wrap errors when adding context.
-   **Variable Naming**:
    -   Use `camelCase` for local variables and function parameters.
    -   Use `PascalCase` for exported identifiers (functions, types, variables).
    -   Keep variable names short but descriptive. Avoid single-letter variable names except for loop counters (`i`, `j`).
-   **Package Naming**:
    -   Package names should be short, concise, and all lowercase.
    -   Avoid using underscores or `camelCase` in package names.

## 2. Code Organization

-   **Package Structure**: Each microservice follows a standard package structure:
    -   `/cmd`: Main application entry points.
    -   `/internal`: Private application and library code.
        -   `/internal/service`: Business logic layer (gRPC service implementations).
        -   `/internal/repository`: Data access layer.
        -   `/internal/model`: Database models (structs).
        -   `/internal/config`: Configuration loading.
    -   `/api`: Protobuf definitions and generated code.
-   **Separation of Concerns**:
    -   Business logic should reside in the `service` layer.
    -   Database and cache interactions should be in the `repository` layer.
    -   HTTP/WebSocket handling and gRPC server setup should be in the `server` or `cmd` packages.

## 3. Commenting

-   **Public APIs**: All exported functions, types, and variables **must** have a doc comment.
    -   The comment should start with the name of the identifier it is describing.
    -   Example: `// UserService is the service for user-related logic.`
-   **Complex Logic**: Add comments to explain complex or non-obvious parts of the code. Explain the "why," not the "what."
-   **TODO Comments**: Use `// TODO:` to mark areas that need future work. Include a brief description of what needs to be done.

## 4. Logging

-   **Library**: We use the custom `clog` library located in `im-infra/clog`.
-   **Structured Logging**: All logs **must** be structured logs. Use key-value pairs to add context.
    -   Example: `logger.Info("User created", clog.String("username", user.Username), clog.Uint64("user_id", user.ID))`
-   **Log Levels**:
    -   `Debug`: For detailed, verbose information useful for debugging.
    -   `Info`: For normal application behavior (e.g., starting a service, handling a request).
    -   `Warn`: For potential issues that do not prevent the application from functioning.
    -   `Error`: For errors that occur but are handled (e.g., a database query fails but is retried).
    -   `Fatal`: For errors that cause the application to crash.

## 5. API and Interface Definitions

-   **Protobuf Style**: Follow the [Google Cloud API Design Guide](https://cloud.google.com/apis/design/style_guide) for `.proto` files.
-   **Clarity and Consistency**: Ensure that API requests and responses are clear, consistent, and well-documented within the `.proto` files.
