package service

import (
	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/im-task/internal/server/grpc"
	"github.com/ceyewan/gochat/im-task/internal/server/kafka"
	"github.com/ceyewan/gochat/pkg/log"
)

type Services struct {
	TaskService        *TaskService
	PersistenceService *PersistenceService
	PushService        *PushService
}

func NewServices(cfg *config.Config, logger *log.Logger, grpcClient *grpc.Client, kafkaProducer *kafka.Producer) *Services {
	persistenceSvc := NewPersistenceService(cfg, logger, grpcClient.GetRepoClient())
	pushSvc := NewPushService(cfg, logger)
	taskSvc := NewTaskService(cfg, logger, grpcClient.GetRepoClient(), kafkaProducer, pushSvc)

	return &Services{
		TaskService:        taskSvc,
		PersistenceService: persistenceSvc,
		PushService:        pushSvc,
	}
}
