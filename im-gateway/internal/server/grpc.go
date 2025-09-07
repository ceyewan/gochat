package server

import (
	"context"
	"errors"
	"time"

	logicv1 "github.com/ceyewan/gochat/api/gen/im_logic/v1"
	"github.com/ceyewan/gochat/im-gateway/internal/config"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCClient gRPC 客户端管理器
type GRPCClient struct {
	config      *config.Config
	coordinator coord.Provider
	logger      clog.Logger

	// gRPC 客户端
	authClient         logicv1.AuthServiceClient
	conversationClient logicv1.ConversationServiceClient
	groupClient        logicv1.GroupServiceClient

	// gRPC 连接
	conn *grpc.ClientConn
}

// NewGRPCClient 创建 gRPC 客户端
func NewGRPCClient(cfg *config.Config, coordinator coord.Provider) (*GRPCClient, error) {
	client := &GRPCClient{
		config:      cfg,
		coordinator: coordinator,
		logger:      clog.Module("grpc-client"),
	}

	// 初始化 gRPC 连接
	if err := client.initConnection(); err != nil {
		return nil, err
	}

	// 初始化客户端
	if err := client.initClients(); err != nil {
		return nil, err
	}

	return client, nil
}

// initConnection 初始化 gRPC 连接
func (gc *GRPCClient) initConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), gc.config.GRPC.ConnTimeout)
	defer cancel()

	var conn *grpc.ClientConn
	var err error

	// 优先使用服务发现
	if gc.config.GRPC.Logic.ServiceName != "" {
		conn, err = gc.coordinator.Registry().GetConnection(ctx, gc.config.GRPC.Logic.ServiceName)
		if err != nil {
			gc.logger.Warn("通过服务发现连接失败，尝试直连", clog.Err(err))
		} else {
			gc.conn = conn
			gc.logger.Info("通过服务发现建立 gRPC 连接", clog.String("service", gc.config.GRPC.Logic.ServiceName))
			return nil
		}
	}

	// 服务发现失败，使用直连地址
	if gc.config.GRPC.Logic.DirectAddr != "" {
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
				grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
			),
		}

		conn, err = grpc.DialContext(ctx, gc.config.GRPC.Logic.DirectAddr, opts...)
		if err != nil {
			return err
		}

		gc.conn = conn
		gc.logger.Info("通过直连地址建立 gRPC 连接", clog.String("address", gc.config.GRPC.Logic.DirectAddr))
		return nil
	}

	return errors.New("未配置 gRPC 服务地址")
}

// initClients 初始化 gRPC 客户端
func (gc *GRPCClient) initClients() error {
	gc.authClient = logicv1.NewAuthServiceClient(gc.conn)
	gc.conversationClient = logicv1.NewConversationServiceClient(gc.conn)
	gc.groupClient = logicv1.NewGroupServiceClient(gc.conn)

	gc.logger.Info("gRPC 客户端初始化完成")
	return nil
}

// GetAuthClient 获取认证服务客户端
func (gc *GRPCClient) GetAuthClient() logicv1.AuthServiceClient {
	return gc.authClient
}

// GetConversationClient 获取会话服务客户端
func (gc *GRPCClient) GetConversationClient() logicv1.ConversationServiceClient {
	return gc.conversationClient
}

// GetGroupClient 获取群组服务客户端
func (gc *GRPCClient) GetGroupClient() logicv1.GroupServiceClient {
	return gc.groupClient
}

// CallWithRetry 带重试的 gRPC 调用
func (gc *GRPCClient) CallWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for i := 0; i <= gc.config.GRPC.Retry.MaxRetries; i++ {
		if i > 0 {
			// 计算延迟时间
			delay := time.Duration(float64(gc.config.GRPC.Retry.InitialDelay) *
				pow(gc.config.GRPC.Retry.BackoffFactor, float64(i-1)))
			if delay > gc.config.GRPC.Retry.MaxDelay {
				delay = gc.config.GRPC.Retry.MaxDelay
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// 执行调用
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		gc.logger.Warn("gRPC 调用失败，准备重试",
			clog.Int("attempt", i+1),
			clog.Int("max_retries", gc.config.GRPC.Retry.MaxRetries),
			clog.Err(err))

		// 检查错误类型，某些错误不需要重试
		if !shouldRetry(err) {
			break
		}
	}

	return lastErr
}

// shouldRetry 判断是否需要重试
func shouldRetry(err error) bool {
	// TODO: 根据错误类型判断是否需要重试
	// 例如：网络错误、超时错误可以重试
	// 但参数错误、权限错误等不应该重试
	return true
}

// Close 关闭 gRPC 连接
func (gc *GRPCClient) Close() error {
	if gc.conn != nil {
		return gc.conn.Close()
	}
	return nil
}

// 辅助函数
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}
