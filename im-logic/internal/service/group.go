package service

import (
	"context"
	"fmt"
	"time"

	logicpb "github.com/ceyewan/gochat/api/gen/im_logic/v1"
	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/server/grpc"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GroupService 群组服务
type GroupService struct {
	config *config.Config
	logger clog.Logger
	client *grpc.Client
}

// NewGroupService 创建群组服务
func NewGroupService(cfg *config.Config, client *grpc.Client) *GroupService {
	logger := clog.Module("group-service")

	return &GroupService{
		config: cfg,
		logger: logger,
		client: client,
	}
}

// CreateGroup 创建群组
func (s *GroupService) CreateGroup(ctx context.Context, req *logicpb.CreateGroupRequest) (*logicpb.CreateGroupResponse, error) {
	s.logger.Info("创建群组",
		clog.String("creator_id", req.CreatorId),
		clog.String("name", req.Name))

	// 验证参数
	if req.CreatorId == "" || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "创建者 ID 和群组名称不能为空")
	}

	// 检查创建权限
	if !s.config.Group.CreatePermission {
		return nil, status.Error(codes.PermissionDenied, "当前不允许创建群组")
	}

	// 生成群组 ID
	groupID := fmt.Sprintf("group-%s", uuid.New().String())

	// 创建群组
	groupRepo := s.client.GetGroupServiceClient()
	createReq := &repopb.CreateGroupRequest{
		GroupId:     groupID,
		Name:        req.Name,
		OwnerId:     req.CreatorId,
		AvatarUrl:   req.AvatarUrl,
		Description: req.Description,
	}

	createResp, err := groupRepo.CreateGroup(ctx, createReq)
	if err != nil {
		s.logger.Error("创建群组失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "创建群组失败")
	}

	// 添加创建者为群主
	memberReq := &repopb.AddGroupMemberRequest{
		GroupId: groupID,
		UserId:  req.CreatorId,
		Role:    int32(logicpb.GroupMemberRole_GROUP_MEMBER_ROLE_OWNER),
	}
	_, err = groupRepo.AddGroupMember(ctx, memberReq)
	if err != nil {
		s.logger.Error("添加群主失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "创建群组失败")
	}

	// 添加初始成员
	for _, memberID := range req.MemberIds {
		if memberID != req.CreatorId { // 跳过创建者
			memberReq := &repopb.AddGroupMemberRequest{
				GroupId: groupID,
				UserId:  memberID,
				Role:    int32(logicpb.GroupMemberRole_GROUP_MEMBER_ROLE_MEMBER),
			}
			_, err = groupRepo.AddGroupMember(ctx, memberReq)
			if err != nil {
				s.logger.Error("添加群组成员失败", clog.String("user_id", memberID), clog.Err(err))
				// 继续添加其他成员
			}
		}
	}

	// 转换群组信息
	group := &logicpb.Group{
		Id:          createResp.Group.Id,
		Name:        createResp.Group.Name,
		AvatarUrl:   createResp.Group.AvatarUrl,
		Description: createResp.Group.Description,
		OwnerId:     createResp.Group.OwnerId,
		MemberCount: createResp.Group.MemberCount,
		CreatedAt:   createResp.Group.CreatedAt,
		UpdatedAt:   createResp.Group.UpdatedAt,
	}

	s.logger.Info("群组创建成功",
		clog.String("group_id", groupID),
		clog.String("creator_id", req.CreatorId))

	return &logicpb.CreateGroupResponse{
		Group: group,
	}, nil
}

// GetGroup 获取群组信息
func (s *GroupService) GetGroup(ctx context.Context, req *logicpb.GetGroupRequest) (*logicpb.GetGroupResponse, error) {
	s.logger.Info("获取群组信息",
		clog.String("user_id", req.UserId),
		clog.String("group_id", req.GroupId))

	// 验证参数
	if req.UserId == "" || req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 和群组 ID 不能为空")
	}

	// 获取群组信息
	groupRepo := s.client.GetGroupServiceClient()
	groupReq := &repopb.GetGroupRequest{
		GroupId: req.GroupId,
	}
	groupResp, err := groupRepo.GetGroup(ctx, groupReq)
	if err != nil {
		s.logger.Error("获取群组信息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "群组不存在")
	}

	// 检查用户是否为群组成员
	// 简化处理，实际需要检查成员表
	userRole := logicpb.GroupMemberRole_GROUP_MEMBER_ROLE_UNSPECIFIED
	if groupResp.Group.OwnerId == req.UserId {
		userRole = logicpb.GroupMemberRole_GROUP_MEMBER_ROLE_OWNER
	}

	// 转换群组信息
	group := &logicpb.Group{
		Id:          groupResp.Group.Id,
		Name:        groupResp.Group.Name,
		AvatarUrl:   groupResp.Group.AvatarUrl,
		Description: groupResp.Group.Description,
		OwnerId:     groupResp.Group.OwnerId,
		MemberCount: groupResp.Group.MemberCount,
		CreatedAt:   groupResp.Group.CreatedAt,
		UpdatedAt:   groupResp.Group.UpdatedAt,
	}

	s.logger.Info("获取群组信息成功",
		clog.String("group_id", req.GroupId))

	return &logicpb.GetGroupResponse{
		Group:    group,
		UserRole: userRole,
	}, nil
}

// JoinGroup 加入群组
func (s *GroupService) JoinGroup(ctx context.Context, req *logicpb.JoinGroupRequest) (*logicpb.JoinGroupResponse, error) {
	s.logger.Info("加入群组",
		clog.String("user_id", req.UserId),
		clog.String("group_id", req.GroupId))

	// 验证参数
	if req.UserId == "" || req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 和群组 ID 不能为空")
	}

	// 获取群组信息
	groupRepo := s.client.GetGroupServiceClient()
	groupReq := &repopb.GetGroupRequest{
		GroupId: req.GroupId,
	}
	groupResp, err := groupRepo.GetGroup(ctx, groupReq)
	if err != nil {
		s.logger.Error("获取群组信息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "群组不存在")
	}

	// 检查群组成员数量限制
	if groupResp.Group.MemberCount >= int32(s.config.Group.MaxMembers) {
		return nil, status.Error(codes.ResourceExhausted, "群组成员已满")
	}

	// 添加成员
	memberReq := &repopb.AddGroupMemberRequest{
		GroupId: req.GroupId,
		UserId:  req.UserId,
		Role:    int32(logicpb.GroupMemberRole_GROUP_MEMBER_ROLE_MEMBER),
	}
	_, err = groupRepo.AddGroupMember(ctx, memberReq)
	if err != nil {
		s.logger.Error("添加群组成员失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "加入群组失败")
	}

	// 转换群组信息
	group := &logicpb.Group{
		Id:          groupResp.Group.Id,
		Name:        groupResp.Group.Name,
		AvatarUrl:   groupResp.Group.AvatarUrl,
		Description: groupResp.Group.Description,
		OwnerId:     groupResp.Group.OwnerId,
		MemberCount: groupResp.Group.MemberCount + 1,
		CreatedAt:   groupResp.Group.CreatedAt,
		UpdatedAt:   time.Now().Unix(),
	}

	s.logger.Info("加入群组成功",
		clog.String("user_id", req.UserId),
		clog.String("group_id", req.GroupId))

	return &logicpb.JoinGroupResponse{
		Success: true,
		Group:   group,
	}, nil
}

// LeaveGroup 离开群组
func (s *GroupService) LeaveGroup(ctx context.Context, req *logicpb.LeaveGroupRequest) (*logicpb.LeaveGroupResponse, error) {
	s.logger.Info("离开群组",
		clog.String("user_id", req.UserId),
		clog.String("group_id", req.GroupId))

	// 验证参数
	if req.UserId == "" || req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 和群组 ID 不能为空")
	}

	// 获取群组信息
	groupRepo := s.client.GetGroupServiceClient()
	groupReq := &repopb.GetGroupRequest{
		GroupId: req.GroupId,
	}
	groupResp, err := groupRepo.GetGroup(ctx, groupReq)
	if err != nil {
		s.logger.Error("获取群组信息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "群组不存在")
	}

	// 群主不能离开群组
	if groupResp.Group.OwnerId == req.UserId {
		return nil, status.Error(codes.FailedPrecondition, "群主不能离开群组，请先转让群主权限")
	}

	// 移除成员
	memberReq := &repopb.RemoveGroupMemberRequest{
		GroupId: req.GroupId,
		UserId:  req.UserId,
	}
	_, err = groupRepo.RemoveGroupMember(ctx, memberReq)
	if err != nil {
		s.logger.Error("移除群组成员失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "离开群组失败")
	}

	s.logger.Info("离开群组成功",
		clog.String("user_id", req.UserId),
		clog.String("group_id", req.GroupId))

	return &logicpb.LeaveGroupResponse{
		Success: true,
	}, nil
}

// GetGroupMembers 获取群组成员列表
func (s *GroupService) GetGroupMembers(ctx context.Context, req *logicpb.GetGroupMembersRequest) (*logicpb.GetGroupMembersResponse, error) {
	s.logger.Info("获取群组成员列表",
		clog.String("user_id", req.UserId),
		clog.String("group_id", req.GroupId))

	// 验证参数
	if req.UserId == "" || req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 和群组 ID 不能为空")
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}

	// 获取群组成员
	groupRepo := s.client.GetGroupServiceClient()
	membersReq := &repopb.GetGroupMembersRequest{
		GroupId:    req.GroupId,
		Offset:     int32((req.Page - 1) * req.PageSize),
		Limit:      int32(req.PageSize),
		RoleFilter: int32(req.RoleFilter),
	}
	membersResp, err := groupRepo.GetGroupMembers(ctx, membersReq)
	if err != nil {
		s.logger.Error("获取群组成员失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取群组成员失败")
	}

	// 获取用户详细信息
	userRepo := s.client.GetUserServiceClient()
	userIDs := make([]string, 0, len(membersResp.Members))
	for _, member := range membersResp.Members {
		userIDs = append(userIDs, member.UserId)
	}

	usersReq := &repopb.GetUsersRequest{
		UserIds: userIDs,
	}
	usersResp, err := userRepo.GetUsers(ctx, usersReq)
	if err != nil {
		s.logger.Error("批量获取用户信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取用户信息失败")
	}

	// 构建用户信息映射
	userMap := make(map[string]*repopb.User)
	for _, user := range usersResp.Users {
		userMap[user.Id] = user
	}

	// 转换成员信息
	members := make([]*logicpb.GroupMember, 0, len(membersResp.Members))
	for _, member := range membersResp.Members {
		userInfo := userMap[member.UserId]
		logicMember := &logicpb.GroupMember{
			UserId:   member.UserId,
			GroupId:  member.GroupId,
			Role:     logicpb.GroupMemberRole(member.Role),
			JoinedAt: member.JoinedAt,
			User: &logicpb.User{
				Id:        userInfo.Id,
				Username:  userInfo.Username,
				Nickname:  userInfo.Nickname,
				AvatarUrl: userInfo.AvatarUrl,
				CreatedAt: userInfo.CreatedAt,
				UpdatedAt: userInfo.UpdatedAt,
			},
		}
		members = append(members, logicMember)
	}

	s.logger.Info("获取群组成员列表成功",
		clog.String("group_id", req.GroupId),
		clog.Int("count", len(members)))

	return &logicpb.GetGroupMembersResponse{
		Members:  members,
		Total:    membersResp.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// UpdateGroup 更新群组信息
func (s *GroupService) UpdateGroup(ctx context.Context, req *logicpb.UpdateGroupRequest) (*logicpb.UpdateGroupResponse, error) {
	s.logger.Info("更新群组信息",
		clog.String("user_id", req.UserId),
		clog.String("group_id", req.GroupId))

	// 验证参数
	if req.UserId == "" || req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 和群组 ID 不能为空")
	}

	// 获取群组信息
	groupRepo := s.client.GetGroupServiceClient()
	groupReq := &repopb.GetGroupRequest{
		GroupId: req.GroupId,
	}
	groupResp, err := groupRepo.GetGroup(ctx, groupReq)
	if err != nil {
		s.logger.Error("获取群组信息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "群组不存在")
	}

	// 检查权限（只有群主和管理员可以更新群组信息）
	if groupResp.Group.OwnerId != req.UserId {
		// TODO: 检查是否为管理员
		return nil, status.Error(codes.PermissionDenied, "无权限更新群组信息")
	}

	// 更新群组信息
	updateReq := &repopb.UpdateGroupRequest{
		GroupId:     req.GroupId,
		Name:        req.Name,
		AvatarUrl:   req.AvatarUrl,
		Description: req.Description,
	}
	_, err = groupRepo.UpdateGroup(ctx, updateReq)
	if err != nil {
		s.logger.Error("更新群组信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "更新群组信息失败")
	}

	// 获取更新后的群组信息
	groupResp, err = groupRepo.GetGroup(ctx, groupReq)
	if err != nil {
		s.logger.Error("获取更新后的群组信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "更新群组信息失败")
	}

	// 转换群组信息
	group := &logicpb.Group{
		Id:          groupResp.Group.Id,
		Name:        groupResp.Group.Name,
		AvatarUrl:   groupResp.Group.AvatarUrl,
		Description: groupResp.Group.Description,
		OwnerId:     groupResp.Group.OwnerId,
		MemberCount: groupResp.Group.MemberCount,
		CreatedAt:   groupResp.Group.CreatedAt,
		UpdatedAt:   groupResp.Group.UpdatedAt,
	}

	s.logger.Info("群组信息更新成功",
		clog.String("group_id", req.GroupId))

	return &logicpb.UpdateGroupResponse{
		Success: true,
		Group:   group,
	}, nil
}

// SetMemberRole 设置成员角色
func (s *GroupService) SetMemberRole(ctx context.Context, req *logicpb.SetMemberRoleRequest) (*logicpb.SetMemberRoleResponse, error) {
	s.logger.Info("设置成员角色",
		clog.String("operator_id", req.OperatorId),
		clog.String("group_id", req.GroupId),
		clog.String("target_user_id", req.TargetUserId))

	// 验证参数
	if req.OperatorId == "" || req.GroupId == "" || req.TargetUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "操作者 ID、群组 ID 和目标用户 ID 不能为空")
	}

	// 获取群组信息
	groupRepo := s.client.GetGroupServiceClient()
	groupReq := &repopb.GetGroupRequest{
		GroupId: req.GroupId,
	}
	groupResp, err := groupRepo.GetGroup(ctx, groupReq)
	if err != nil {
		s.logger.Error("获取群组信息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "群组不存在")
	}

	// 检查权限（只有群主可以设置成员角色）
	if groupResp.Group.OwnerId != req.OperatorId {
		return nil, status.Error(codes.PermissionDenied, "无权限设置成员角色")
	}

	// 验证角色
	if req.NewRole == logicpb.GroupMemberRole_GROUP_MEMBER_ROLE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "无效的角色")
	}

	// 更新成员角色
	updateReq := &repopb.UpdateMemberRoleRequest{
		GroupId: req.GroupId,
		UserId:  req.TargetUserId,
		NewRole: int32(req.NewRole),
	}
	_, err = groupRepo.UpdateMemberRole(ctx, updateReq)
	if err != nil {
		s.logger.Error("更新成员角色失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "设置成员角色失败")
	}

	s.logger.Info("成员角色设置成功",
		clog.String("group_id", req.GroupId),
		clog.String("target_user_id", req.TargetUserId))

	return &logicpb.SetMemberRoleResponse{
		Success: true,
	}, nil
}

// RemoveMember 移除成员
func (s *GroupService) RemoveMember(ctx context.Context, req *logicpb.RemoveMemberRequest) (*logicpb.RemoveMemberResponse, error) {
	s.logger.Info("移除成员",
		clog.String("operator_id", req.OperatorId),
		clog.String("group_id", req.GroupId),
		clog.String("target_user_id", req.TargetUserId))

	// 验证参数
	if req.OperatorId == "" || req.GroupId == "" || req.TargetUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "操作者 ID、群组 ID 和目标用户 ID 不能为空")
	}

	// 获取群组信息
	groupRepo := s.client.GetGroupServiceClient()
	groupReq := &repopb.GetGroupRequest{
		GroupId: req.GroupId,
	}
	groupResp, err := groupRepo.GetGroup(ctx, groupReq)
	if err != nil {
		s.logger.Error("获取群组信息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "群组不存在")
	}

	// 检查权限（只有群主和管理员可以移除成员）
	if groupResp.Group.OwnerId != req.OperatorId {
		// TODO: 检查是否为管理员
		return nil, status.Error(codes.PermissionDenied, "无权限移除成员")
	}

	// 不能移除群主
	if groupResp.Group.OwnerId == req.TargetUserId {
		return nil, status.Error(codes.FailedPrecondition, "不能移除群主")
	}

	// 移除成员
	memberReq := &repopb.RemoveGroupMemberRequest{
		GroupId: req.GroupId,
		UserId:  req.TargetUserId,
	}
	_, err = groupRepo.RemoveGroupMember(ctx, memberReq)
	if err != nil {
		s.logger.Error("移除成员失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "移除成员失败")
	}

	s.logger.Info("成员移除成功",
		clog.String("group_id", req.GroupId),
		clog.String("target_user_id", req.TargetUserId))

	return &logicpb.RemoveMemberResponse{
		Success: true,
	}, nil
}
