package server

import (
	"context"

	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/config"
	"github.com/ceyewan/gochat/im-repo/internal/repository"
	"github.com/ceyewan/gochat/im-repo/internal/service"
	"google.golang.org/grpc"
)

// Server 定义 im-repo 服务器接口
type Server interface {
	// RegisterServices 注册 gRPC 服务
	RegisterServices(grpcServer *grpc.Server)

	// Shutdown 优雅关闭服务器
	Shutdown(ctx context.Context) error

	// GetHealthChecker 获取健康检查器
	GetHealthChecker() HealthChecker
}

// server 服务器实现
type server struct {
	config *config.Config
	logger clog.Logger

	// 数据库和缓存
	database *repository.Database
	cache    *repository.CacheManager

	// 数据仓储
	userRepo         *repository.UserRepository
	messageRepo      *repository.MessageRepository
	conversationRepo *repository.ConversationRepository
	groupRepo        *repository.GroupRepository
	onlineStatusRepo *repository.OnlineStatusRepository

	// gRPC 服务
	userService         *service.UserService
	messageService      *service.MessageService
	conversationService *service.ConversationService
	groupService        *service.GroupService
	onlineStatusService *service.OnlineStatusService
}

// New 创建新的服务器实例
func New(cfg *config.Config) (Server, error) {
	logger := clog.Module("repo-server")

	s := &server{
		config: cfg,
		logger: logger,
	}

	// 1. 初始化数据库连接
	database, err := repository.NewDatabase(cfg)
	if err != nil {
		logger.Error("初始化数据库失败", clog.Err(err))
		return nil, err
	}
	s.database = database

	// 2. 初始化缓存连接
	cache, err := repository.NewCacheManager(cfg)
	if err != nil {
		logger.Error("初始化缓存失败", clog.Err(err))
		return nil, err
	}
	s.cache = cache

	// 3. 创建数据仓储实例
	s.userRepo = repository.NewUserRepository(database, cache)
	s.messageRepo = repository.NewMessageRepository(database, cache)
	s.conversationRepo = repository.NewConversationRepository(database, cache)
	s.groupRepo = repository.NewGroupRepository(database, cache)
	s.onlineStatusRepo = repository.NewOnlineStatusRepository(cache)

	// 4. 执行数据库迁移
	ctx := context.Background()
	if err := database.Migrate(ctx); err != nil {
		logger.Error("数据库迁移失败", clog.Err(err))
		return nil, err
	}

	// 5. 创建 gRPC 服务实例
	s.userService = service.NewUserService(s.userRepo)
	s.messageService = service.NewMessageService(s.messageRepo, s.conversationRepo)
	s.conversationService = service.NewConversationService(s.conversationRepo, s.messageRepo)
	s.groupService = service.NewGroupService(s.groupRepo)
	s.onlineStatusService = service.NewOnlineStatusService(s.onlineStatusRepo)

	logger.Info("im-repo 服务器创建成功")
	return s, nil
}

// GetHealthChecker 获取健康检查器
func (s *server) GetHealthChecker() HealthChecker {
	return NewHealthChecker(s)
}

// RegisterServices 注册 gRPC 服务
func (s *server) RegisterServices(grpcServer *grpc.Server) {
	s.logger.Info("注册 gRPC 服务...")

	// 注册具体的 gRPC 服务
	repopb.RegisterUserServiceServer(grpcServer, s.userService)
	repopb.RegisterMessageServiceServer(grpcServer, s.messageService)
	repopb.RegisterConversationServiceServer(grpcServer, s.conversationService)
	repopb.RegisterGroupServiceServer(grpcServer, s.groupService)
	repopb.RegisterOnlineStatusServiceServer(grpcServer, s.onlineStatusService)

	s.logger.Info("gRPC 服务注册完成")
}

// Shutdown 优雅关闭服务器
func (s *server) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭 im-repo 服务器...")

	// 1. 关闭缓存连接
	if s.cache != nil {
		if err := s.cache.Close(); err != nil {
			s.logger.Error("关闭缓存连接失败", clog.Err(err))
		}
	}

	// 2. 关闭数据库连接
	if s.database != nil {
		if err := s.database.Close(); err != nil {
			s.logger.Error("关闭数据库连接失败", clog.Err(err))
		}
	}

	// TODO: 3. 服务注销

	s.logger.Info("im-repo 服务器已关闭")
	return nil
}
