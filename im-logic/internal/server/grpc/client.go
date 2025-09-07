package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Client gRPC 客户端
type Client struct {
	config           *config.Config
	logger           clog.Logger
	conn             *grpc.ClientConn
	userRepo         repob.UserServiceClient
	messageRepo      repob.MessageServiceClient
	conversationRepo repob.ConversationServiceClient
	groupRepo        repob.GroupServiceClient
	onlineStatusRepo repob.OnlineStatusServiceClient
}

// NewClient 创建 gRPC 客户端
func NewClient(cfg *config.Config) (*Client, error) {
	logger := clog.Module("grpc-client")

	client := &Client{
		config: cfg,
		logger: logger,
	}

	// 创建连接
	if err := client.createConnection(); err != nil {
		return nil, err
	}

	// 创建服务客户端
	client.createServiceClients()

	logger.Info("gRPC 客户端创建成功", clog.String("address", cfg.Repo.GRPC.Address))
	return client, nil
}

// createConnection 创建 gRPC 连接
func (c *Client) createConnection() error {
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  1.0 * time.Second,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   120 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		),
	}

	// 配置 TLS
	if c.config.Repo.GRPC.EnableTLS {
		// TODO: 实现 TLS 配置
		// creds, err := credentials.LoadTLSCredentials()
		// if err != nil {
		//     return err
		// }
		// opts = append(opts, grpc.WithTransportCredentials(creds))
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// 创建连接
	conn, err := grpc.Dial(
		c.config.Repo.GRPC.Address,
		opts...,
	)
	if err != nil {
		c.logger.Error("创建 gRPC 连接失败", clog.Err(err))
		return fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	c.conn = conn

	// 启动连接状态监控
	go c.monitorConnection()

	return nil
}

// createServiceClients 创建服务客户端
func (c *Client) createServiceClients() {
	c.userRepo = repob.NewUserServiceClient(c.conn)
	c.messageRepo = repob.NewMessageServiceClient(c.conn)
	c.conversationRepo = repob.NewConversationServiceClient(c.conn)
	c.groupRepo = repob.NewGroupServiceClient(c.conn)
	c.onlineStatusRepo = repob.NewOnlineStatusServiceClient(c.conn)
}

// monitorConnection 监控连接状态
func (c *Client) monitorConnection() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		state := c.conn.GetState()
		c.logger.Debug("gRPC 连接状态", clog.String("state", state.String()))

		// 如果连接断开，尝试重连
		if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			c.logger.Warn("gRPC 连接断开，尝试重连...")
			if err := c.reconnect(); err != nil {
				c.logger.Error("重连失败", clog.Err(err))
			}
		}
	}
}

// reconnect 重连
func (c *Client) reconnect() error {
	// 关闭旧连接
	if c.conn != nil {
		c.conn.Close()
	}

	// 创建新连接
	return c.createConnection()
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetUserServiceClient 获取用户服务客户端
func (c *Client) GetUserServiceClient() repob.UserServiceClient {
	return c.userRepo
}

// GetMessageServiceClient 获取消息服务客户端
func (c *Client) GetMessageServiceClient() repob.MessageServiceClient {
	return c.messageRepo
}

// GetConversationServiceClient 获取会话服务客户端
func (c *Client) GetConversationServiceClient() repob.ConversationServiceClient {
	return c.conversationRepo
}

// GetGroupServiceClient 获取群组服务客户端
func (c *Client) GetGroupServiceClient() repob.GroupServiceClient {
	return c.groupRepo
}

// GetOnlineStatusServiceClient 获取在线状态服务客户端
func (c *Client) GetOnlineStatusServiceClient() repob.OnlineStatusServiceClient {
	return c.onlineStatusRepo
}

// WithTimeout 创建带超时的上下文
func (c *Client) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(c.config.Repo.GRPC.Timeout)*time.Second)
}

// RetryWithBackoff 带退避的重试机制
func (c *Client) RetryWithBackoff(ctx context.Context, fn func(ctx context.Context) error) error {
	var lastErr error

	for i := 0; i < c.config.Repo.GRPC.MaxRetries; i++ {
		// 创建带超时的上下文
		timeoutCtx, cancel := c.WithTimeout(ctx)
		defer cancel()

		err := fn(timeoutCtx)
		if err == nil {
			return nil
		}

		lastErr = err
		c.logger.Warn("操作失败，准备重试", clog.Int("attempt", i+1), clog.Err(err))

		// 计算退避时间
		backoffTime := time.Duration(c.config.Repo.GRPC.RetryInterval) * time.Second
		for j := 0; j < i; j++ {
			backoffTime *= 2
		}

		// 等待退避时间
		select {
		case <-time.After(backoffTime):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("重试 %d 次后仍然失败: %w", c.config.Repo.GRPC.MaxRetries, lastErr)
}
