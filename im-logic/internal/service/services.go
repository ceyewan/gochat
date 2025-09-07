package service

import (
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/server/grpc"
)

// Services 所有业务服务的集合
type Services struct {
	Auth         *AuthService
	Conversation *ConversationService
	Message      *MessageService
	Group        *GroupService
}

// NewServices 创建所有业务服务
func NewServices(cfg *config.Config, client *grpc.Client) *Services {
	return &Services{
		Auth:         NewAuthService(cfg, client),
		Conversation: NewConversationService(cfg, client),
		Message:      NewMessageService(cfg, client),
		Group:        NewGroupService(cfg, client),
	}
}
