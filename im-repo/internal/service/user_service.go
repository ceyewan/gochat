package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/model"
	"github.com/ceyewan/gochat/im-repo/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserService 用户服务实现
type UserService struct {
	repopb.UnimplementedUserServiceServer
	userRepo *repository.UserRepository
	logger   clog.Logger
}

// NewUserService 创建用户服务
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   clog.Module("user-service"),
	}
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, req *repopb.CreateUserRequest) (*repopb.CreateUserResponse, error) {
	s.logger.Info("创建用户请求", clog.String("username", req.Username))

	// 参数验证
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "用户名不能为空")
	}
	if req.PasswordHash == "" {
		return nil, status.Error(codes.InvalidArgument, "密码不能为空")
	}

	// 创建用户模型
	user := &model.User{
		Username:     req.Username,
		PasswordHash: req.PasswordHash,
		Nickname:     req.Nickname,
		AvatarURL:    req.AvatarUrl,
	}

	// 如果昵称为空，使用用户名作为昵称
	if user.Nickname == "" {
		user.Nickname = user.Username
	}

	// 创建用户
	err := s.userRepo.CreateUser(ctx, user)
	if err != nil {
		s.logger.Error("创建用户失败", clog.Err(err))
		// 检查是否是用户名重复错误
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, status.Error(codes.AlreadyExists, "用户名已存在")
		}
		return nil, status.Error(codes.Internal, "创建用户失败")
	}

	// 构造响应
	resp := &repopb.CreateUserResponse{
		User: s.modelToProto(user),
	}

	s.logger.Info("用户创建成功",
		clog.String("username", req.Username),
		clog.String("user_id", fmt.Sprintf("%d", user.ID)))

	return resp, nil
}

// GetUser 获取用户信息
func (s *UserService) GetUser(ctx context.Context, req *repopb.GetUserRequest) (*repopb.GetUserResponse, error) {
	s.logger.Debug("获取用户信息请求", clog.String("user_id", req.UserId))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 转换用户ID
	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("获取用户信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取用户信息失败")
	}

	if user == nil {
		return nil, status.Error(codes.NotFound, "用户不存在")
	}

	// 构造响应
	resp := &repopb.GetUserResponse{
		User: s.modelToProto(user),
	}

	return resp, nil
}

// GetUserByUsername 根据用户名获取用户信息
func (s *UserService) GetUserByUsername(ctx context.Context, req *repopb.GetUserByUsernameRequest) (*repopb.GetUserByUsernameResponse, error) {
	s.logger.Debug("根据用户名获取用户信息请求", clog.String("username", req.Username))

	// 参数验证
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "用户名不能为空")
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Error("根据用户名获取用户信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取用户信息失败")
	}

	if user == nil {
		return nil, status.Error(codes.NotFound, "用户不存在")
	}

	// 构造响应
	resp := &repopb.GetUserByUsernameResponse{
		User: s.modelToProto(user),
	}

	return resp, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, req *repopb.UpdateUserRequest) (*repopb.UpdateUserResponse, error) {
	s.logger.Info("更新用户信息请求", clog.String("user_id", req.UserId))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 转换用户ID
	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	// 构建更新字段
	updates := make(map[string]interface{})

	if req.Nickname != nil {
		updates["nickname"] = req.Nickname.Value
	}
	if req.AvatarUrl != nil {
		updates["avatar_url"] = req.AvatarUrl.Value
	}
	if req.PasswordHash != nil {
		updates["password_hash"] = req.PasswordHash.Value
	}

	// 如果没有要更新的字段
	if len(updates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "没有要更新的字段")
	}

	// 更新用户信息
	err = s.userRepo.UpdateUser(ctx, userID, updates)
	if err != nil {
		s.logger.Error("更新用户信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "更新用户信息失败")
	}

	// 获取更新后的用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("获取更新后的用户信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取更新后的用户信息失败")
	}

	// 构造响应
	resp := &repopb.UpdateUserResponse{
		User: s.modelToProto(user),
	}

	s.logger.Info("用户信息更新成功", clog.String("user_id", req.UserId))
	return resp, nil
}

// BatchGetUsers 批量获取用户信息
func (s *UserService) BatchGetUsers(ctx context.Context, req *repopb.BatchGetUsersRequest) (*repopb.BatchGetUsersResponse, error) {
	s.logger.Debug("批量获取用户信息请求", clog.Int("user_count", len(req.UserIds)))

	// 参数验证
	if len(req.UserIds) == 0 {
		return &repopb.BatchGetUsersResponse{
			Users: make(map[string]*repopb.User),
		}, nil
	}

	// 转换用户ID列表
	userIDs := make([]uint64, len(req.UserIds))
	for i, userIDStr := range req.UserIds {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			s.logger.Error("用户ID格式错误",
				clog.String("user_id", userIDStr),
				clog.Err(err))
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("用户ID格式错误: %s", userIDStr))
		}
		userIDs[i] = userID
	}

	// 批量获取用户信息
	users, err := s.userRepo.BatchGetUsers(ctx, userIDs)
	if err != nil {
		s.logger.Error("批量获取用户信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "批量获取用户信息失败")
	}

	// 转换为 protobuf 格式
	protoUsers := make(map[string]*repopb.User)
	for userID, user := range users {
		protoUsers[fmt.Sprintf("%d", userID)] = s.modelToProto(user)
	}

	// 构造响应
	resp := &repopb.BatchGetUsersResponse{
		Users: protoUsers,
	}

	s.logger.Debug("批量获取用户信息成功",
		clog.Int("requested", len(req.UserIds)),
		clog.Int("found", len(protoUsers)))

	return resp, nil
}

// VerifyPassword 验证用户密码
func (s *UserService) VerifyPassword(ctx context.Context, req *repopb.VerifyPasswordRequest) (*repopb.VerifyPasswordResponse, error) {
	s.logger.Debug("验证用户密码请求", clog.String("username", req.Username))

	// 参数验证
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "用户名不能为空")
	}
	if req.PasswordHash == "" {
		return nil, status.Error(codes.InvalidArgument, "密码不能为空")
	}

	// 验证密码
	user, valid, err := s.userRepo.VerifyPassword(ctx, req.Username, req.PasswordHash)
	if err != nil {
		s.logger.Error("验证用户密码失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "验证密码失败")
	}

	var protoUser *repopb.User
	if user != nil {
		protoUser = s.modelToProto(user)
	}

	// 构造响应
	resp := &repopb.VerifyPasswordResponse{
		Valid: valid,
		User:  protoUser,
	}

	s.logger.Debug("用户密码验证完成",
		clog.String("username", req.Username),
		clog.Bool("valid", valid))

	return resp, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, req *repopb.DeleteUserRequest) (*repopb.DeleteUserResponse, error) {
	s.logger.Info("删除用户请求", clog.String("user_id", req.UserId))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 转换用户ID
	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	// 删除用户
	err = s.userRepo.DeleteUser(ctx, userID)
	if err != nil {
		s.logger.Error("删除用户失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "删除用户失败")
	}

	// 构造响应
	resp := &repopb.DeleteUserResponse{
		Success: true,
	}

	s.logger.Info("用户删除成功", clog.String("user_id", req.UserId))
	return resp, nil
}

// modelToProto 将模型转换为 protobuf 格式
func (s *UserService) modelToProto(user *model.User) *repopb.User {
	return &repopb.User{
		Id:           fmt.Sprintf("%d", user.ID),
		Username:     user.Username,
		Nickname:     user.Nickname,
		AvatarUrl:    user.AvatarURL,
		CreatedAt:    user.CreatedAt.Unix(),
		UpdatedAt:    user.UpdatedAt.Unix(),
		PasswordHash: user.PasswordHash, // 注意：在实际生产环境中，不应该返回密码哈希
	}
}
