package service

import (
	"context"
	"fmt"
	"strconv"

	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/model"
	"github.com/ceyewan/gochat/im-repo/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GroupService 群组服务实现
type GroupService struct {
	repopb.UnimplementedGroupServiceServer
	groupRepo *repository.GroupRepository
	logger    clog.Logger
}

// NewGroupService 创建群组服务
func NewGroupService(groupRepo *repository.GroupRepository) *GroupService {
	return &GroupService{
		groupRepo: groupRepo,
		logger:    clog.Namespace("group-service"),
	}
}

// CreateGroup 创建群组
func (s *GroupService) CreateGroup(ctx context.Context, req *repopb.CreateGroupRequest) (*repopb.CreateGroupResponse, error) {
	s.logger.Info("创建群组请求",
		clog.String("name", req.Name),
		clog.String("owner_id", req.OwnerId))

	// 参数验证
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "群组名称不能为空")
	}
	if req.OwnerId == "" {
		return nil, status.Error(codes.InvalidArgument, "群主ID不能为空")
	}

	// 转换群主ID
	ownerID, err := strconv.ParseUint(req.OwnerId, 10, 64)
	if err != nil {
		s.logger.Error("群主ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "群主ID格式错误")
	}

	// 创建群组模型
	group := &model.Group{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     ownerID,
	}

	// 创建群组
	err = s.groupRepo.CreateGroup(ctx, group)
	if err != nil {
		s.logger.Error("创建群组失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "创建群组失败")
	}

	// 构造响应
	resp := &repopb.CreateGroupResponse{
		Group: s.modelToProto(group),
	}

	s.logger.Info("群组创建成功",
		clog.String("group_id", fmt.Sprintf("%d", group.ID)),
		clog.String("name", req.Name))

	return resp, nil
}

// GetGroup 获取群组信息
func (s *GroupService) GetGroup(ctx context.Context, req *repopb.GetGroupRequest) (*repopb.GetGroupResponse, error) {
	s.logger.Debug("获取群组信息请求", clog.String("group_id", req.GroupId))

	// 参数验证
	if req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群组ID不能为空")
	}

	// 转换群组ID
	groupID, err := strconv.ParseUint(req.GroupId, 10, 64)
	if err != nil {
		s.logger.Error("群组ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "群组ID格式错误")
	}

	// 获取群组信息
	group, err := s.groupRepo.GetGroup(ctx, groupID)
	if err != nil {
		s.logger.Error("获取群组信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取群组信息失败")
	}

	if group == nil {
		return nil, status.Error(codes.NotFound, "群组不存在")
	}

	// 构造响应
	resp := &repopb.GetGroupResponse{
		Group: s.modelToProto(group),
	}

	return resp, nil
}

// UpdateGroup 更新群组信息
func (s *GroupService) UpdateGroup(ctx context.Context, req *repopb.UpdateGroupRequest) (*repopb.UpdateGroupResponse, error) {
	s.logger.Info("更新群组信息请求", clog.String("group_id", req.GroupId))

	// 参数验证
	if req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群组ID不能为空")
	}

	// 转换群组ID
	groupID, err := strconv.ParseUint(req.GroupId, 10, 64)
	if err != nil {
		s.logger.Error("群组ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "群组ID格式错误")
	}

	// 暂不支持更新
	s.logger.Warn("UpdateGroup 方法未实现")
	return nil, status.Error(codes.Unimplemented, "方法未实现")

	// 获取更新后的群组信息
	updatedGroup, err := s.groupRepo.GetGroup(ctx, groupID)
	if err != nil {
		s.logger.Error("获取更新后的群组信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取更新后的群组信息失败")
	}

	// 构造响应
	resp := &repopb.UpdateGroupResponse{
		Group: s.modelToProto(updatedGroup),
	}

	s.logger.Info("群组信息更新成功", clog.String("group_id", req.GroupId))

	return resp, nil
}

// AddGroupMember 添加群组成员
func (s *GroupService) AddGroupMember(ctx context.Context, req *repopb.AddGroupMemberRequest) (*repopb.AddGroupMemberResponse, error) {
	s.logger.Info("添加群组成员请求",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId),
		clog.Int32("role", req.Role))

	// 参数验证
	if req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群组ID不能为空")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 转换ID
	groupID, err := strconv.ParseUint(req.GroupId, 10, 64)
	if err != nil {
		s.logger.Error("群组ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "群组ID格式错误")
	}

	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	role := int(req.Role)
	if role <= 0 {
		role = 1 // 默认普通成员
	}

	// 添加群组成员
	err = s.groupRepo.AddGroupMember(ctx, groupID, userID, role)
	if err != nil {
		s.logger.Error("添加群组成员失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "添加群组成员失败")
	}

	// 构造响应
	resp := &repopb.AddGroupMemberResponse{
		Member: &repopb.GroupMember{
			GroupId: req.GroupId,
			UserId:  req.UserId,
			Role:    req.Role,
		},
	}

	s.logger.Info("群组成员添加成功",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId))

	return resp, nil
}

// RemoveGroupMember 移除群组成员
func (s *GroupService) RemoveGroupMember(ctx context.Context, req *repopb.RemoveGroupMemberRequest) (*repopb.RemoveGroupMemberResponse, error) {
	s.logger.Info("移除群组成员请求",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId))

	// 参数验证
	if req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群组ID不能为空")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 转换ID
	groupID, err := strconv.ParseUint(req.GroupId, 10, 64)
	if err != nil {
		s.logger.Error("群组ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "群组ID格式错误")
	}

	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	// 移除群组成员
	err = s.groupRepo.RemoveGroupMember(ctx, groupID, userID)
	if err != nil {
		s.logger.Error("移除群组成员失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "移除群组成员失败")
	}

	// 构造响应
	resp := &repopb.RemoveGroupMemberResponse{
		Success: true,
	}

	s.logger.Info("群组成员移除成功",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId))

	return resp, nil
}

// GetGroupMembers 获取群组成员列表
func (s *GroupService) GetGroupMembers(ctx context.Context, req *repopb.GetGroupMembersRequest) (*repopb.GetGroupMembersResponse, error) {
	s.logger.Debug("获取群组成员列表请求",
		clog.String("group_id", req.GroupId),
		clog.Int32("offset", req.Offset),
		clog.Int32("limit", req.Limit))

	// 参数验证
	if req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群组ID不能为空")
	}

	// 转换群组ID
	groupID, err := strconv.ParseUint(req.GroupId, 10, 64)
	if err != nil {
		s.logger.Error("群组ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "群组ID格式错误")
	}

	offset := int(req.Offset)
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20 // 默认限制
	}
	if limit > 100 {
		limit = 100 // 最大限制
	}

	// 获取群组成员列表
	members, total, hasMore, err := s.groupRepo.GetGroupMembers(ctx, groupID, offset, limit, int(req.RoleFilter))
	if err != nil {
		s.logger.Error("获取群组成员列表失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取群组成员列表失败")
	}

	// 转换为 protobuf 格式
	protoMembers := make([]*repopb.GroupMember, len(members))
	for i, member := range members {
		protoMembers[i] = s.memberModelToProto(member)
	}

	// 构造响应
	resp := &repopb.GetGroupMembersResponse{
		Members: protoMembers,
		Total:   total,
		HasMore: hasMore,
	}

	s.logger.Debug("获取群组成员列表成功",
		clog.String("group_id", req.GroupId),
		clog.Int64("total", total),
		clog.Int("returned", len(members)))

	return resp, nil
}

// GetMembersOnlineStatus 批量获取成员在线状态
func (s *GroupService) GetMembersOnlineStatus(ctx context.Context, req *repopb.GetMembersOnlineStatusRequest) (*repopb.GetMembersOnlineStatusResponse, error) {
	s.logger.Debug("批量获取成员在线状态请求", clog.Int("user_count", len(req.UserIds)))

	// 这里简化实现，实际项目中需要调用在线状态服务
	s.logger.Warn("GetMembersOnlineStatus 方法需要实现")

	// 构造响应
	resp := &repopb.GetMembersOnlineStatusResponse{
		Statuses: []*repopb.OnlineStatus{}, // 返回空列表作为占位符
	}

	return resp, nil
}

// UpdateMemberRole 更新成员角色
func (s *GroupService) UpdateMemberRole(ctx context.Context, req *repopb.UpdateMemberRoleRequest) (*repopb.UpdateMemberRoleResponse, error) {
	s.logger.Info("更新成员角色请求",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId),
		clog.Int32("new_role", req.NewRole))

	// 参数验证
	if req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群组ID不能为空")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}
	if req.NewRole <= 0 {
		return nil, status.Error(codes.InvalidArgument, "角色值必须大于0")
	}

	// 转换ID
	groupID, err := strconv.ParseUint(req.GroupId, 10, 64)
	if err != nil {
		s.logger.Error("群组ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "群组ID格式错误")
	}

	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	// 更新成员角色
	err = s.groupRepo.UpdateMemberRole(ctx, groupID, userID, int(req.NewRole))
	if err != nil {
		s.logger.Error("更新成员角色失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "更新成员角色失败")
	}

	// 构造响应
	resp := &repopb.UpdateMemberRoleResponse{
		Success: true,
	}

	s.logger.Info("成员角色更新成功",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId),
		clog.Int32("new_role", req.NewRole))

	return resp, nil
}

// modelToProto 将群组模型转换为 protobuf 格式
func (s *GroupService) modelToProto(group *model.Group) *repopb.Group {
	if group == nil {
		return nil
	}
	return &repopb.Group{
		Id:          fmt.Sprintf("%d", group.ID),
		Name:        group.Name,
		Description: group.Description,
		OwnerId:     fmt.Sprintf("%d", group.OwnerID),
		MemberCount: int32(group.MemberCount),
		CreatedAt:   group.CreatedAt.Unix(),
		UpdatedAt:   group.UpdatedAt.Unix(),
	}
}

// memberModelToProto 将群组成员模型转换为 protobuf 格式
func (s *GroupService) memberModelToProto(member *model.GroupMember) *repopb.GroupMember {
	return &repopb.GroupMember{
		GroupId:  fmt.Sprintf("%d", member.GroupID),
		UserId:   fmt.Sprintf("%d", member.UserID),
		Role:     int32(member.Role),
		JoinedAt: member.JoinedAt.Unix(),
	}
}
