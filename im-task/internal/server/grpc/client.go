package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/pkg/log"
	"github.com/ceyewan/gochat/pkg/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
)

type Client struct {
	config     *config.Config
	logger     *log.Logger
	conn       *grpc.ClientConn
	repoClient repobb.ImRepoServiceClient
}

func NewClient(cfg *config.Config, logger *log.Logger) (*Client, error) {
	c := &Client{
		config: cfg,
		logger: logger,
	}

	if err := c.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC services: %w", err)
	}

	return c, nil
}

func (c *Client) connect() error {
	target := fmt.Sprintf("%s:%d", c.config.RepoService.Host, c.config.RepoService.Port)

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Grpc.Client.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  c.config.Grpc.Client.Retry.Backoff,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   c.config.Grpc.Client.Retry.MaxBackoff,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024),
			grpc.MaxCallSendMsgSize(4*1024*1024),
		),
		grpc.WithUnaryInterceptor(wrappers.UnaryClientInterceptor(c.logger)),
		grpc.WithStreamInterceptor(wrappers.StreamClientInterceptor(c.logger)),
	)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %w", target, err)
	}

	c.conn = conn
	c.repoClient = repobb.NewImRepoServiceClient(conn)

	c.logger.Info("Connected to gRPC services", "target", target)
	return nil
}

func (c *Client) GetRepoClient() repobb.ImRepoServiceClient {
	return c.repoClient
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) HealthCheck() bool {
	if c.conn == nil {
		return false
	}
	return c.conn.GetState() == connectivity.Ready
}

func (c *Client) Reconnect() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return c.connect()
}
