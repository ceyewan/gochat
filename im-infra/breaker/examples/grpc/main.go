package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/breaker"
	"github.com/ceyewan/gochat/im-infra/clog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// BreakerClientInterceptor 创建一个 gRPC 客户端拦截器，集成熔断器功能
func BreakerClientInterceptor(provider breaker.Provider) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 使用 gRPC 的方法名作为熔断器的名称，为每个方法创建独立的熔断器
		b := provider.GetBreaker(method)

		// 将真正的 gRPC 调用包裹在熔断器的 Do 方法中
		err := b.Do(ctx, func() error {
			return invoker(ctx, method, req, reply, cc, opts...)
		})

		// 将熔断器错误转换为标准的 gRPC 错误码
		if err == breaker.ErrBreakerOpen {
			return status.Error(codes.Unavailable, err.Error())
		}
		return err
	}
}

// ExampleUserService 定义一个示例用户服务接口
type ExampleUserServiceClient interface {
	GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error)
}

type GetUserRequest struct {
	UserID string `json:"user_id"`
}

type GetUserResponse struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

// mockUserServiceClient 模拟用户服务客户端
type mockUserServiceClient struct {
	conn *grpc.ClientConn
}

func (m *mockUserServiceClient) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	// 模拟服务调用
	return &GetUserResponse{
		UserID: req.UserID,
		Name:   "John Doe",
		Email:  "john@example.com",
	}, nil
}

func main() {
	// 初始化日志器
	logger := clog.Namespace("grpc-breaker-example")

	// 创建 breaker 配置
	config := breaker.GetDefaultConfig("grpc-client", "development")

	// 创建 breaker Provider
	breakerProvider, err := breaker.New(context.Background(), config,
		breaker.WithLogger(logger),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer breakerProvider.Close()

	// 创建 gRPC 连接（这里使用模拟地址）
	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(BreakerClientInterceptor(breakerProvider)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// 创建服务客户端
	client := &mockUserServiceClient{conn: conn}

	// 执行多次调用，演示熔断器在 gRPC 调用中的保护作用
	for i := 1; i <= 5; i++ {
		fmt.Printf("gRPC call attempt %d: ", i)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		resp, err := client.GetUser(ctx, &GetUserRequest{UserID: fmt.Sprintf("user-%d", i)})
		if err != nil {
			if status.Code(err) == codes.Unavailable {
				fmt.Println("❌ Circuit breaker OPEN - gRPC call blocked")
			} else {
				fmt.Printf("❌ gRPC call failed: %v\n", err)
			}
		} else {
			fmt.Printf("✅ gRPC call succeeded: %+v\n", resp)
		}

		time.Sleep(300 * time.Millisecond)
	}

	fmt.Println("\nGRPC breaker example completed. The circuit breaker protects against service unavailability.")
}