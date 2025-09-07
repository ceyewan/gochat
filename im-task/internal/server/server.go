package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/im-task/internal/server/grpc"
	"github.com/ceyewan/gochat/im-task/internal/server/kafka"
	"github.com/ceyewan/gochat/im-task/internal/service"
	"github.com/ceyewan/gochat/pkg/log"
	"github.com/ceyewan/gochat/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	config        *config.Config
	logger        *log.Logger
	grpcClient    *grpc.Client
	kafkaProducer *kafka.Producer
	kafkaConsumer *kafka.Consumer
	services      *service.Services
	httpServer    *http.Server
}

func NewServer(cfg *config.Config, logger *log.Logger) (*Server, error) {
	s := &Server{
		config: cfg,
		logger: logger,
	}

	if err := s.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize server: %w", err)
	}

	return s, nil
}

func (s *Server) initialize() error {
	if err := s.initGRPCClient(); err != nil {
		return fmt.Errorf("failed to initialize gRPC client: %w", err)
	}

	if err := s.initKafka(); err != nil {
		return fmt.Errorf("failed to initialize Kafka: %w", err)
	}

	s.initServices()
	s.initHTTPServer()

	return nil
}

func (s *Server) initGRPCClient() error {
	grpcClient, err := grpc.NewClient(s.config, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}

	s.grpcClient = grpcClient
	return nil
}

func (s *Server) initKafka() error {
	kafkaProducer, err := kafka.NewProducer(s.config, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	s.kafkaProducer = kafkaProducer

	s.services = service.NewServices(s.config, s.logger, s.grpcClient, kafkaProducer)

	kafkaConsumer, err := kafka.NewConsumer(s.config, s.logger, s.services.TaskService, s.services.PersistenceService)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	s.kafkaConsumer = kafkaConsumer
	return nil
}

func (s *Server) initServices() {
}

func (s *Server) initHTTPServer() {
	mux := http.NewServeMux()

	if s.config.Metrics.Enabled {
		mux.Handle(s.config.Metrics.Path, promhttp.Handler())
	}

	if s.config.Health.Enabled {
		mux.HandleFunc(s.config.Health.Path, s.healthHandler)
	}

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Metrics.Port),
		Handler: middleware.Logging(s.logger)(mux),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting im-task server",
		"version", s.config.Server.Version,
		"port", s.config.Server.Port,
	)

	if err := s.kafkaConsumer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start Kafka consumer: %w", err)
	}

	go func() {
		s.logger.Info("Starting HTTP server", "port", s.config.Metrics.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	s.logger.Info("im-task server started successfully")
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping im-task server")

	if err := s.kafkaConsumer.Close(); err != nil {
		s.logger.Error("Failed to close Kafka consumer", "error", err)
	}

	if err := s.kafkaProducer.Close(); err != nil {
		s.logger.Error("Failed to close Kafka producer", "error", err)
	}

	if err := s.grpcClient.Close(); err != nil {
		s.logger.Error("Failed to close gRPC client", "error", err)
	}

	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Failed to shutdown HTTP server", "error", err)
		}
	}

	s.logger.Info("im-task server stopped")
	return nil
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"server":    s.config.Server.Name,
		"version":   s.config.Server.Version,
		"timestamp": time.Now().Unix(),
	}

	if s.grpcClient != nil {
		health["grpc_client"] = map[string]interface{}{
			"connected": s.grpcClient.HealthCheck(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

func (s *Server) WaitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Stop(ctx); err != nil {
		s.logger.Error("Server shutdown error", "error", err)
	}
}
