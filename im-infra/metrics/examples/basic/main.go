package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. Configure and initialize the metrics provider.
	cfg := metrics.DefaultConfig()
	cfg.ServiceName = "my-awesome-service"
	cfg.PrometheusListenAddr = ":9091" // Expose metrics on this port
	cfg.ExporterType = "stdout"        // For demo, print traces to console

	provider, err := metrics.New(cfg)
	if err != nil {
		log.Fatalf("failed to create metrics provider: %v", err)
	}

	// 2. Defer the shutdown of the provider.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		// Give a bit of time for metrics to be exported
		time.Sleep(5 * time.Second)
		if err := provider.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown metrics provider: %v", err)
		}
	}()

	// 3. Create a gRPC server with the interceptor.
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			provider.GRPCServerInterceptor(),
		),
	)

	// In a real app, you would register your gRPC services here.
	// For this example, we'll just enable reflection so tools can see the server.
	reflection.Register(server)

	// 4. Start the gRPC server.
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	go func() {
		log.Println("gRPC server listening on :8081")
		if err := server.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	log.Println("Service is running. Press Ctrl+C to exit.")
	log.Println("Metrics available at http://localhost:9091/metrics")
	log.Println("Any gRPC call to localhost:8081 will be traced and measured (e.g., using grpcurl).")

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	server.GracefulStop()
	log.Println("Server stopped.")
}
