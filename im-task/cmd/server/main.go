package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/im-task/internal/server"
	"github.com/ceyewan/gochat/pkg/log"
)

func main() {
	var (
		configPath = flag.String("config", "configs/config.yaml", "Path to config file")
		version    = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *version {
		fmt.Println("im-task v1.0.0")
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := log.NewLogger(cfg.Log)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting im-task service",
		"version", cfg.Server.Version,
		"config", *configPath,
	)

	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("Server shutdown completed")
}
