# GoChat gRPC 接口规范

**版本**: 1.0  
**日期**: 2025-07-20  
**作者**: AI架构师

## 1. 概述

本文档定义了GoChat系统中所有服务间的gRPC接口规范，确保服务间通信的一致性和可维护性。

## 2. 通用规范

### 2.1 命名规范
- 服务名称：使用PascalCase，如 `UserService`
- 方法名称：使用PascalCase，如 `GetUser`
- 消息名称：使用PascalCase，如 `GetUserRequest`
- 字段名称：使用snake_case，如 `user_id`

### 2.2 错误处理
- 使用标准gRPC状态码
- 错误详情使用google.rpc.ErrorInfo
- 关键错误需要包含trace_id用于问题追踪

### 2.3 通用字段
```protobuf
// 通用请求头
message RequestHeader {
  string trace_id = 1;        // 链路追踪ID
  string user_id = 2;         // 请求用户ID
  int64 timestamp = 3;        // 请求时间戳
}

// 通用响应头
message ResponseHeader {
  string trace_id = 1;        // 链路追踪ID
  int32 code = 2;             // 业务错误码
  string message = 3;         // 错误消息
}
```

## 3. im-logic 对外接口

### 3.1 认证服务 (AuthService)
```protobuf
service AuthService {
  // 用户登录
  rpc Login(LoginRequest) returns (LoginResponse);
  // 用户注册
  rpc Register(RegisterRequest) returns (RegisterResponse);
  // 游客登录
  rpc GuestLogin(GuestLoginRequest) returns (GuestLoginResponse);
  // 验证Token
  rpc VerifyToken(VerifyTokenRequest) returns (VerifyTokenResponse);
}

message LoginRequest {
  RequestHeader header = 1;
  string username = 2;
  string password = 3;
}

message LoginResponse {
  ResponseHeader header = 1;
  string token = 2;
  User user = 3;
}

message User {
  string user_id = 1;
  string username = 2;
  string nickname = 3;
  string avatar_url = 4;
  bool is_guest = 5;
  int64 created_at = 6;
}
```

### 3.2 会话服务 (ConversationService)
```protobuf
service ConversationService {
  // 获取用户会话列表
  rpc GetConversations(GetConversationsRequest) returns (GetConversationsResponse);
  // 创建私聊会话
  rpc CreatePrivateConversation(CreatePrivateConversationRequest) returns (CreatePrivateConversationResponse);
  // 标记会话已读
  rpc MarkConversationAsRead(MarkAsReadRequest) returns (google.protobuf.Empty);
  // 获取会话信息
  rpc GetConversationInfo(GetConversationInfoRequest) returns (GetConversationInfoResponse);
}

message GetConversationsRequest {
  RequestHeader header = 1;
  string user_id = 2;
}

message GetConversationsResponse {
  ResponseHeader header = 1;
  repeated Conversation conversations = 2;
}

message Conversation {
  string conversation_id = 1;
  string type = 2;              // single/group/world
  ConversationTarget target = 3;
  string last_message = 4;
  int64 last_message_time = 5;
  int64 last_message_seq = 6;
  int32 unread_count = 7;
}

message ConversationTarget {
  // 单聊字段
  string user_id = 1;
  string username = 2;
  // 群聊字段
  string group_id = 3;
  string group_name = 4;
  int32 member_count = 5;
  // 通用字段
  string avatar_url = 6;
  string description = 7;
}
```

### 3.3 消息服务 (MessageService)
```protobuf
service MessageService {
  // 获取会话消息
  rpc GetMessages(GetMessagesRequest) returns (GetMessagesResponse);
  // 发送消息 (HTTP API使用)
  rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);
}

message GetMessagesRequest {
  RequestHeader header = 1;
  string conversation_id = 2;
  int32 page = 3;
  int32 size = 4;
  int64 before_seq = 5;         // 获取指定序列号之前的消息
}

message GetMessagesResponse {
  ResponseHeader header = 1;
  repeated Message messages = 2;
  Pagination pagination = 3;
}

message Message {
  string message_id = 1;
  string conversation_id = 2;
  string sender_id = 3;
  string sender_name = 4;
  string content = 5;
  int32 message_type = 6;       // 1:文本, 2:图片, 3:文件
  int64 seq_id = 7;
  int64 send_time = 8;
  string client_msg_id = 9;
}

message Pagination {
  int32 page = 1;
  int32 size = 2;
  int64 total = 3;
  bool has_more = 4;
}
```

### 3.4 用户服务 (UserService)
```protobuf
service UserService {
  // 获取用户信息
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  // 搜索用户
  rpc SearchUser(SearchUserRequest) returns (SearchUserResponse);
  // 获取用户在线状态
  rpc GetUserOnlineStatus(GetUserOnlineStatusRequest) returns (GetUserOnlineStatusResponse);
}

message GetUserRequest {
  RequestHeader header = 1;
  string user_id = 2;
}

message GetUserResponse {
  ResponseHeader header = 1;
  User user = 2;
}
```

### 3.5 群组服务 (GroupService)
```protobuf
service GroupService {
  // 创建群组
  rpc CreateGroup(CreateGroupRequest) returns (CreateGroupResponse);
  // 获取群组信息
  rpc GetGroup(GetGroupRequest) returns (GetGroupResponse);
  // 获取群组成员
  rpc GetGroupMembers(GetGroupMembersRequest) returns (GetGroupMembersResponse);
  // 加入群组
  rpc JoinGroup(JoinGroupRequest) returns (google.protobuf.Empty);
  // 离开群组
  rpc LeaveGroup(LeaveGroupRequest) returns (google.protobuf.Empty);
}

message CreateGroupRequest {
  RequestHeader header = 1;
  string group_name = 2;
  repeated string member_ids = 3;
}

message CreateGroupResponse {
  ResponseHeader header = 1;
  Group group = 2;
  Conversation conversation = 3;
}

message Group {
  string group_id = 1;
  string group_name = 2;
  string owner_id = 3;
  string avatar_url = 4;
  int32 member_count = 5;
  int64 created_at = 6;
}
```

## 4. im-repo 对外接口

### 4.1 用户仓储 (UserRepo)
```protobuf
service UserRepo {
  // 创建用户
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  // 根据ID获取用户
  rpc GetUserById(GetUserByIdRequest) returns (GetUserByIdResponse);
  // 根据用户名获取用户
  rpc GetUserByUsername(GetUserByUsernameRequest) returns (GetUserByUsernameResponse);
  // 更新用户信息
  rpc UpdateUser(UpdateUserRequest) returns (google.protobuf.Empty);
  // 获取用户会话状态
  rpc GetUserSession(GetUserSessionRequest) returns (GetUserSessionResponse);
  // 设置用户会话状态
  rpc SetUserSession(SetUserSessionRequest) returns (google.protobuf.Empty);
}

message CreateUserRequest {
  RequestHeader header = 1;
  string username = 2;
  string password_hash = 3;
  string nickname = 4;
  bool is_guest = 5;
}

message GetUserSessionRequest {
  RequestHeader header = 1;
  string user_id = 2;
}

message GetUserSessionResponse {
  ResponseHeader header = 1;
  string gateway_id = 2;
  int64 login_at = 3;
  bool is_online = 4;
}
```

### 4.2 消息仓储 (MessageRepo)
```protobuf
service MessageRepo {
  // 保存消息
  rpc SaveMessage(SaveMessageRequest) returns (google.protobuf.Empty);
  // 获取消息
  rpc GetMessages(GetMessagesRequest) returns (GetMessagesResponse);
  // 获取会话序列号
  rpc GetConversationSeq(GetConversationSeqRequest) returns (GetConversationSeqResponse);
  // 增加会话序列号
  rpc IncrConversationSeq(IncrConversationSeqRequest) returns (IncrConversationSeqResponse);
}

message SaveMessageRequest {
  RequestHeader header = 1;
  string message_id = 2;
  string conversation_id = 3;
  string sender_id = 4;
  string content = 5;
  int32 message_type = 6;
  int64 seq_id = 7;
  string client_msg_id = 8;
}
```

### 4.3 群组仓储 (GroupRepo)
```protobuf
service GroupRepo {
  // 创建群组
  rpc CreateGroup(CreateGroupRequest) returns (CreateGroupResponse);
  // 获取群组信息
  rpc GetGroup(GetGroupRequest) returns (GetGroupResponse);
  // 获取群组成员
  rpc GetGroupMembers(GetGroupMembersRequest) returns (GetGroupMembersResponse);
  // 获取在线群组成员
  rpc GetOnlineGroupMembers(GetOnlineGroupMembersRequest) returns (GetOnlineGroupMembersResponse);
  // 添加群组成员
  rpc AddGroupMember(AddGroupMemberRequest) returns (google.protobuf.Empty);
  // 移除群组成员
  rpc RemoveGroupMember(RemoveGroupMemberRequest) returns (google.protobuf.Empty);
}

message GetOnlineGroupMembersRequest {
  RequestHeader header = 1;
  string group_id = 2;
}

message GetOnlineGroupMembersResponse {
  ResponseHeader header = 1;
  repeated OnlineGroupMember members = 2;
}

message OnlineGroupMember {
  string user_id = 1;
  string gateway_id = 2;
}
```

## 5. Kafka 消息格式

### 5.1 通用消息格式
```protobuf
message KafkaMessage {
  string trace_id = 1;
  map<string, string> headers = 2;
  google.protobuf.Any body = 3;
}
```

### 5.2 上行消息 (im-upstream-topic)
```protobuf
message UpstreamMessage {
  string user_id = 1;
  string conversation_id = 2;
  string content = 3;
  int32 message_type = 4;
  string client_msg_id = 5;
  int64 timestamp = 6;
}
```

### 5.3 下行消息 (im-downstream-topic-{gateway_id})
```protobuf
message DownstreamMessage {
  string target_user_id = 1;
  Message message = 2;
}
```

### 5.4 任务消息 (im-task-topic)
```protobuf
message TaskMessage {
  string task_type = 1;         // large_group_fanout, offline_push等
  google.protobuf.Any payload = 2;
  int32 retry_count = 3;
  int64 created_at = 4;
}

message LargeGroupFanoutTask {
  string group_id = 1;
  string message_id = 2;
  int32 batch_size = 3;         // 分批处理大小
}
```

## 6. 接口版本管理

### 6.1 版本策略
- 使用语义化版本号 (v1.0.0)
- 向后兼容的变更增加小版本号
- 破坏性变更增加主版本号
- 同时支持多个版本，逐步迁移

### 6.2 接口演进
- 新增字段使用optional标记
- 废弃字段使用deprecated标记
- 提供接口变更文档和迁移指南

## 7. 测试规范

### 7.1 单元测试
- 每个gRPC方法都需要单元测试
- 测试覆盖率要求达到80%以上
- 包含正常流程和异常流程测试

### 7.2 集成测试
- 服务间接口的集成测试
- 使用testcontainers进行环境隔离
- 自动化测试流水线集成

## 8. 监控与观测

### 8.1 指标监控
- gRPC调用次数、延迟、错误率
- 服务间依赖关系监控
- 接口性能基线和SLA监控

### 8.2 链路追踪
- 所有gRPC调用自动注入trace_id
- 跨服务调用链路可视化
- 性能瓶颈分析和优化建议
