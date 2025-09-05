package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/model"
	"github.com/ceyewan/gochat/im-repo/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GroupService 群组服务实现
type GroupService struct {
	v1.UnimplementedGroupServiceServer
	groupRepo *repository.GroupRepository
	logger    clog.Logger
}

// NewGroupService 创建群组服务
func NewGroupService(groupRepo *repository.GroupRepository) *GroupService {
	return &GroupService{
		groupRepo: groupRepo,
		logger:    clog.Module("group-service"),
	}
}

// CreateGroup 创建群组
func (s *GroupService) CreateGroup(ctx context.Context, req *v1.CreateGroupRequest) (*v1.CreateGroupResponse, error) {
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
		GroupType:   int(req.GroupType),
		MaxMembers:  int(req.MaxMembers),
		Avatar:      req.Avatar,
	}

	// 设置默认值
	if group.MaxMembers <= 0 {
		group.MaxMembers = 500 // 默认最大成员数
	}
	if group.GroupType <= 0 {
		group.GroupType = 1 // 默认普通群组
	}

	// 创建群组
	err = s.groupRepo.CreateGroup(ctx, group)
	if err != nil {
		s.logger.Error("创建群组失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "创建群组失败")
	}

	// 构造响应
	resp := &v1.CreateGroupResponse{
		Group: s.modelToProto(group),
	}

	s.logger.Info("群组创建成功",
		clog.String("group_id", fmt.Sprintf("%d", group.ID)),
		clog.String("name", req.Name))

	return resp, nil
}

// GetGroup 获取群组信息
func (s *GroupService) GetGroup(ctx context.Context, req *v1.GetGroupRequest) (*v1.GetGroupResponse, error) {
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
	resp := &v1.GetGroupResponse{
		Group: s.modelToProto(group),
	}

	return resp, nil
}

// AddGroupMember 添加群组成员
func (s *GroupService) AddGroupMember(ctx context.Context, req *v1.AddGroupMemberRequest) (*v1.AddGroupMemberResponse, error) {
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
	resp := &v1.AddGroupMemberResponse{
		Success: true,
	}

	s.logger.Info("群组成员添加成功",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId))

	return resp, nil
}

// RemoveGroupMember 移除群组成员
func (s *GroupService) RemoveGroupMember(ctx context.Context, req *v1.RemoveGroupMemberRequest) (*v1.RemoveGroupMemberResponse, error) {
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
	resp := &v1.RemoveGroupMemberResponse{
		Success: true,
	}

	s.logger.Info("群组成员移除成功",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId))

	return resp, nil
}

// GetGroupMembers 获取群组成员列表
func (s *GroupService) GetGroupMembers(ctx context.Context, req *v1.GetGroupMembersRequest) (*v1.GetGroupMembersResponse, error) {
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
	protoMembers := make([]*v1.GroupMember, len(members))
	for i, member := range members {
		protoMembers[i] = s.memberModelToProto(member)
	}

	// 构造响应
	resp := &v1.GetGroupMembersResponse{
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

// UpdateMemberRole 更新成员角色
func (s *GroupService) UpdateMemberRole(ctx context.Context, req *v1.UpdateMemberRoleRequest) (*v1.UpdateMemberRoleResponse, error) {
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
	resp := &v1.UpdateMemberRoleResponse{
		Success: true,
	}

	s.logger.Info("成员角色更新成功",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId),
		clog.Int32("new_role", req.NewRole))

	return resp, nil
}

// IsGroupMember 检查用户是否为群组成员
func (s *GroupService) IsGroupMember(ctx context.Context, req *v1.IsGroupMemberRequest) (*v1.IsGroupMemberResponse, error) {
	s.logger.Debug("检查群组成员请求",
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

	// 检查是否为群组成员
	isMember, role, err := s.groupRepo.IsGroupMember(ctx, groupID, userID)
	if err != nil {
		s.logger.Error("检查群组成员失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "检查群组成员失败")
	}

	// 构造响应
	resp := &v1.IsGroupMemberResponse{
		IsMember: isMember,
		Role:     int32(role),
	}

	s.logger.Debug("群组成员检查完成",
		clog.String("group_id", req.GroupId),
		clog.String("user_id", req.UserId),
		clog.Bool("is_member", isMember),
		clog.Int("role", role))

	return resp, nil
}

// BatchGetGroupMembers 批量获取群组成员
func (s *GroupService) BatchGetGroupMembers(ctx context.Context, req *v1.BatchGetGroupMembersRequest) (*v1.BatchGetGroupMembersResponse, error) {
	s.logger.Debug("批量获取群组成员请求", clog.Int("group_count", len(req.GroupIds)))

	// 参数验证
	if len(req.GroupIds) == 0 {
		return &v1.BatchGetGroupMembersResponse{
			GroupMembers: make(map[string]*v1.GroupMemberList),
		}, nil
	}

	result := make(map[string]*v1.GroupMemberList)

	// 为每个群组获取成员列表
	for _, groupIDStr := range req.GroupIds {
		groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
		if err != nil {
			s.logger.Error("群组ID格式错误",
				clog.String("group_id", groupIDStr),
				clog.Err(err))
			continue // 跳过格式错误的ID
		}

		// 获取群组成员（限制数量以避免返回过多数据）
		members, _, _, err := s.groupRepo.GetGroupMembers(ctx, groupID, 0, 1000, 0)
		if err != nil {
			s.logger.Error("获取群组成员失败",
				clog.String("group_id", groupIDStr),
				clog.Err(err))
			continue // 跳过失败的群组
		}

		// 转换为 protobuf 格式
		protoMembers := make([]*v1.GroupMember, len(members))
		for i, member := range members {
			protoMembers[i] = s.memberModelToProto(member)
		}

		result[groupIDStr] = &v1.GroupMemberList{
			Members: protoMembers,
		}
	}

	// 构造响应
	resp := &v1.BatchGetGroupMembersResponse{
		GroupMembers: result,
	}

	s.logger.Debug("批量获取群组成员完成",
		clog.Int("requested_groups", len(req.GroupIds)),
		clog.Int("successful_groups", len(result)))

	return resp, nil
}

// modelToProto 将群组模型转换为 protobuf 格式
func (s *GroupService) modelToProto(group *model.Group) *v1.Group {
	return &v1.Group{
		Id:          fmt.Sprintf("%d", group.ID),
		Name:        group.Name,
		Description: group.Description,
		OwnerId:     fmt.Sprintf("%d", group.OwnerID),
		GroupType:   int32(group.GroupType),
		MemberCount: int32(group.MemberCount),
		MaxMembers:  int32(group.MaxMembers),
		Avatar:      group.Avatar,
		CreatedAt:   group.CreatedAt.Unix(),
		UpdatedAt:   group.UpdatedAt.Unix(),
	}
}

// memberModelToProto 将群组成员模型转换为 protobuf 格式
func (s *GroupService) memberModelToProto(member *model.GroupMember) *v1.GroupMember {
	return &v1.GroupMember{
		GroupId:  fmt.Sprintf("%d", member.GroupID),
		UserId:   fmt.Sprintf("%d", member.UserID),
		Role:     int32(member.Role),
		JoinedAt: member.JoinedAt.Unix(),
	}
}
