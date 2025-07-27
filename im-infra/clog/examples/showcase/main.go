package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// showcase an opinionated and comprehensive guide to using the clog library.
// It demonstrates best practices for logging in a typical Go application,
// covering everything from basic usage to advanced configuration and error handling.
func main() {
	// --- Section 1: Basic Logging ---
	// Demonstrates the simplest way to use clog with its default configuration.
	// The default setup is developer-friendly: human-readable console output,
	// info level, with color and caller information.
	fmt.Println("--- 1. Basic Logging (Default Configuration) ---")
	clog.Info("Service is starting up...", clog.String("version", "1.0.1"))
	clog.Warn("Configuration file not found, using default settings.", clog.String("config_path", "./config.yaml"))
	// Debug logs are ignored by default (default level is info).
	clog.Debug("This is a debug message, it won't be visible.")

	// --- Section 2: Structured Logging with Fields ---
	// Shows how to add structured context to your logs using various field types.
	// This is crucial for making logs machine-readable and queryable in production.
	fmt.Println("\n--- 2. Structured Logging with Various Field Types ---")
	clog.Info("User profile updated",
		clog.Int("userID", 12345),
		clog.String("username", "alice"),
		clog.Bool("is_active", true),
		clog.Float64("rating", 4.8),
		clog.Duration("login_duration", 2*time.Hour+30*time.Minute),
		clog.Time("last_seen", time.Now()),
		clog.Strings("roles", []string{"admin", "editor"}),
	)

	// --- Section 3: Contextual Logging with TraceID ---
	// The recommended way for logging within request handlers or business logic.
	// `clog.C(ctx)` automatically extracts a "traceID" (or similar keys)
	// from the context, linking all logs for a single operation.
	fmt.Println("\n--- 3. Contextual Logging (Recommended Practice) ---")
	ctx := context.WithValue(context.Background(), "traceID", "trace-abc-123")
	clog.C(ctx).Info("Processing incoming request", clog.String("http_method", "POST"))

	// --- Section 4: Modular Logging ---
	// `clog.Module()` creates a logger for a specific component (e.g., a package or service).
	// This helps to categorize logs and identify their origin.
	// For performance, it's best to create module loggers once and reuse them.
	fmt.Println("\n--- 4. Modular Logging for Different Components ---")
	userServiceLogger := clog.Module("user-service")
	paymentServiceLogger := clog.Module("payment-service")

	userServiceLogger.Info("User service initialized.")
	paymentServiceLogger.Info("Payment service initialized.")

	// --- Section 5: Efficient Logging with `With` ---
	// `With()` creates a new logger with pre-set fields. This is highly efficient
	// as the fields are only marshaled once. Ideal for adding constant context
	// like service name, version, or request-specific data.
	fmt.Println("\n--- 5. Efficient Logging with `With` for Constant Context ---")
	requestLogger := clog.C(ctx).With(
		clog.String("service", "api-gateway"),
		clog.String("requestID", "req-xyz-789"),
	)
	requestLogger.Info("Request validation successful.")
	requestLogger.Info("Forwarding request to upstream.")

	// --- Section 6: Advanced Error Handling ---
	// Demonstrates how to properly log errors. `clog.Err()` is a helper for `zap.Error()`.
	// It ensures errors are displayed correctly, including stack traces for wrapped errors.
	fmt.Println("\n--- 6. Advanced Error Handling ---")
	dbErr := errors.New("connection refused")
	repoErr := fmt.Errorf("failed to query user: %w", dbErr)
	requestLogger.Error("Failed to handle request", clog.Err(repoErr))

	// --- Section 7: Custom Logger Configuration (Production Setup) ---
	// Shows how to create and initialize a logger with a custom configuration,
	// which is typical for a production environment.
	// Here, we configure JSON output, file rotation, and a debug log level.
	fmt.Println("\n--- 7. Custom Logger for Production (JSON to a file) ---")
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		clog.Error("Failed to create log directory", clog.Err(err))
		return
	}
	logFile := filepath.Join(logDir, "app.log")

	prodConfig := clog.Config{
		Level:  "debug", // Log everything from debug up
		Format: "json",  // Structured JSON for machines
		Output: logFile, // Write to a file
		Rotation: &clog.RotationConfig{
			MaxSize:    1,    // 1 MB
			MaxBackups: 3,    // Keep 3 old files
			MaxAge:     7,    // Keep files for 7 days
			Compress:   true, // Compress rotated files
		},
	}

	// It's good practice to initialize a dedicated logger for the application
	// instead of relying solely on the global one.
	prodLogger, err := clog.New(prodConfig)
	if err != nil {
		clog.Error("Failed to create production logger", clog.Err(err))
		return
	}

	prodLogger.Info("Production logger initialized.", clog.String("log_path", logFile))
	prodLogger.Debug("This debug message will now be visible in the log file.")
	fmt.Printf("Production logs are being written to %s. Please check the file.\n", logFile)

	fmt.Println("\n--- Showcase Complete ---")
}
