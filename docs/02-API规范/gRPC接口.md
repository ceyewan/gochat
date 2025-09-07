# gRPC 接口规范

## 概述

GoChat 系统内部服务间通信使用 gRPC 协议。本文档详细描述了各个服务的 gRPC 接口定义。

## 服务列表

### 1. LogicService (im-logic)

负责处理业务逻辑，包括用户认证、消息处理、会话管理等。

### 2. RepoService (im-repo)

负责数据持久化和查询，包括用户、消息、会话、群组等数据管理。

## 基础类型定义

### 错误类型

```protobuf
enum ErrorCode {
    ERROR_CODE_NONE = 0;
    ERROR_CODE_INVALID_PARAMS = 1001;
    ERROR_CODE_UNAUTHORIZED = 1002;
    ERROR_CODE_PERMISSION_DENIED = 1003;
    ERROR_CODE_NOT_FOUND = 1004;
    ERROR_CODE_INTERNAL_ERROR = 1005;
    ERROR_CODE_DUPLICATE_OPERATION = 1006;
    ERROR_CODE_RATE_LIMIT = 1007;
}

message Error {
    ErrorCode code = 1;
    string message = 2;
    map<string, string> details = 3;
}
```

### 分页类型

```protobuf
message PaginationRequest {
    int32 page = 1;
    int32 limit = 2;
}

message PaginationResponse {
    int32 total = 1;
    int32 page = 2;
    int32 limit = 3;
}
```

### 用户类型

```protobuf
message User {
    string user_id = 1;
    string username = 2;
    string email = 3;
    string nickname = 4;
    string avatar = 5;
    int32 status = 6;
    int64 created_at = 7;
    int64 updated_at = 8;
}

message UserProfile {
    string user_id = 1;
    string username = 2;
    string nickname = 3;
    string avatar = 4;
    string bio = 5;
    int32 status = 6;
    int64 created_at = 7;
    int64 updated_at = 8;
}
```

### 消息类型

```protobuf
message Message {
    string message_id = 1;
    string conversation_id = 2;
    string from_user_id = 3;
    string to_user_id = 4;
    string content = 5;
    string message_type = 6;
    int32 status = 7;
    map<string, string> metadata = 8;
    int64 created_at = 9;
    int64 updated_at = 10;
}

message MessageStatus {
    string message_id = 1;
    int32 status = 2;
    int64 timestamp = 3;
}
```

### 会话类型

```protobuf
message Conversation {
    string conversation_id = 1;
    string type = 2;
    string name = 3;
    string avatar = 4;
    string last_message = 5;
    int64 created_at = 6;
    int64 updated_at = 7;
}

message ConversationInfo {
    string conversation_id = 1;
    string type = 2;
    string name = 3;
    string avatar = 4;
    Message last_message = 5;
    int32 unread_count = 6;
    int64 updated_at = 7;
}
```

### 群组类型

```protobuf
message Group {
    string group_id = 1;
    string name = 2;
    string description = 3;
    string avatar = 4;
    string owner_id = 5;
    int32 max_members = 6;
    int32 current_members = 7;
    int64 created_at = 8;
    int64 updated_at = 9;
}

message GroupMember {
    string user_id = 1;
    string nickname = 2;
    string avatar = 3;
    string role = 4;
    int64 joined_at = 5;
}
```

## LogicService 接口定义

### 认证相关

#### Authenticate

```protobuf
message AuthRequest {
    string username = 1;
    string password = 2;
    string device_id = 3;
}

message AuthResponse {
    bool success = 1;
    string access_token = 2;
    string refresh_token = 3;
    int64 expires_in = 4;
    UserProfile user_info = 5;
    Error error = 6;
}
```

#### ValidateToken

```protobuf
message ValidateTokenRequest {
    string token = 1;
}

message ValidateTokenResponse {
    bool valid = 1;
    string user_id = 2;
    int64 expires_at = 3;
    Error error = 4;
}
```

### 用户相关

#### GetUserProfile

```protobuf
message GetUserProfileRequest {
    string user_id = 1;
}

message GetUserProfileResponse {
    UserProfile profile = 1;
    Error error = 2;
}
```

#### UpdateUserProfile

```protobuf
message UpdateUserProfileRequest {
    string user_id = 1;
    string nickname = 2;
    string avatar = 3;
    string bio = 4;
}

message UpdateUserProfileResponse {
    bool success = 1;
    UserProfile profile = 2;
    Error error = 3;
}
```

#### SearchUsers

```protobuf
message SearchUsersRequest {
    string query = 1;
    PaginationRequest pagination = 2;
}

message SearchUsersResponse {
    repeated User users = 1;
    PaginationResponse pagination = 2;
    Error error = 3;
}
```

### 消息相关

#### SendMessage

```protobuf
message SendMessageRequest {
    string from_user_id = 1;
    string conversation_id = 2;
    string to_user_id = 3;
    string content = 4;
    string message_type = 5;
    map<string, string> metadata = 6;
}

message SendMessageResponse {
    string message_id = 1;
    int64 created_at = 2;
    Error error = 3;
}
```

#### GetMessageHistory

```protobuf
message GetMessageHistoryRequest {
    string conversation_id = 1;
    string before_message_id = 2;
    int32 limit = 3;
}

message GetMessageHistoryResponse {
    repeated Message messages = 1;
    bool has_more = 2;
    Error error = 3;
}
```

#### MarkMessageRead

```protobuf
message MarkMessageReadRequest {
    string user_id = 1;
    string conversation_id = 2;
    string message_id = 3;
}

message MarkMessageReadResponse {
    bool success = 1;
    Error error = 2;
}
```

#### GetUnreadMessages

```protobuf
message GetUnreadMessagesRequest {
    string user_id = 1;
}

message GetUnreadMessagesResponse {
    int32 total_count = 1;
    repeated ConversationInfo conversations = 2;
    Error error = 3;
}
```

### 会话相关

#### CreateConversation

```protobuf
message CreateConversationRequest {
    string type = 1;
    string name = 2;
    repeated string user_ids = 3;
}

message CreateConversationResponse {
    string conversation_id = 1;
    Conversation conversation = 2;
    Error error = 3;
}
```

#### GetConversations

```protobuf
message GetConversationsRequest {
    string user_id = 1;
    PaginationRequest pagination = 2;
}

message GetConversationsResponse {
    repeated ConversationInfo conversations = 1;
    PaginationResponse pagination = 2;
    Error error = 3;
}
```

#### UpdateConversation

```protobuf
message UpdateConversationRequest {
    string conversation_id = 1;
    string name = 2;
    string avatar = 3;
}

message UpdateConversationResponse {
    bool success = 1;
    Conversation conversation = 2;
    Error error = 3;
}
```

#### DeleteConversation

```protobuf
message DeleteConversationRequest {
    string conversation_id = 1;
    string user_id = 2;
}

message DeleteConversationResponse {
    bool success = 1;
    Error error = 2;
}
```

### 群组相关

#### CreateGroup

```protobuf
message CreateGroupRequest {
    string name = 1;
    string description = 2;
    string avatar = 3;
    string owner_id = 4;
    int32 max_members = 5;
    repeated string member_ids = 6;
}

message CreateGroupResponse {
    string group_id = 1;
    Group group = 2;
    Error error = 3;
}
```

#### GetGroupInfo

```protobuf
message GetGroupInfoRequest {
    string group_id = 1;
}

message GetGroupInfoResponse {
    Group group = 1;
    Error error = 2;
}
```

#### UpdateGroup

```protobuf
message UpdateGroupRequest {
    string group_id = 1;
    string name = 2;
    string description = 3;
    string avatar = 4;
    int32 max_members = 5;
}

message UpdateGroupResponse {
    bool success = 1;
    Group group = 2;
    Error error = 3;
}
```

#### JoinGroup

```protobuf
message JoinGroupRequest {
    string group_id = 1;
    string user_id = 2;
}

message JoinGroupResponse {
    bool success = 1;
    Error error = 2;
}
```

#### LeaveGroup

```protobuf
message LeaveGroupRequest {
    string group_id = 1;
    string user_id = 2;
}

message LeaveGroupResponse {
    bool success = 1;
    Error error = 2;
}
```

#### GetGroupMembers

```protobuf
message GetGroupMembersRequest {
    string group_id = 1;
    PaginationRequest pagination = 2;
}

message GetGroupMembersResponse {
    repeated GroupMember members = 1;
    PaginationResponse pagination = 2;
    Error error = 3;
}
```

#### AddGroupMember

```protobuf
message AddGroupMemberRequest {
    string group_id = 1;
    repeated string user_ids = 2;
    string operator_id = 3;
}

message AddGroupMemberResponse {
    bool success = 1;
    repeated string added_user_ids = 2;
    Error error = 3;
}
```

#### RemoveGroupMember

```protobuf
message RemoveGroupMemberRequest {
    string group_id = 1;
    string user_id = 2;
    string operator_id = 3;
}

message RemoveGroupMemberResponse {
    bool success = 1;
    Error error = 2;
}
```

## RepoService 接口定义

### 用户相关

#### CreateUser

```protobuf
message CreateUserRequest {
    string username = 1;
    string email = 2;
    string password = 3;
    string nickname = 4;
}

message CreateUserResponse {
    string user_id = 1;
    User user = 2;
    Error error = 3;
}
```

#### GetUser

```protobuf
message GetUserRequest {
    string user_id = 1;
}

message GetUserResponse {
    User user = 1;
    Error error = 2;
}
```

#### UpdateUser

```protobuf
message UpdateUserRequest {
    string user_id = 1;
    string nickname = 2;
    string avatar = 3;
    int32 status = 4;
}

message UpdateUserResponse {
    bool success = 1;
    User user = 2;
    Error error = 3;
}
```

#### GetUsers

```protobuf
message GetUsersRequest {
    repeated string user_ids = 1;
}

message GetUsersResponse {
    repeated User users = 1;
    Error error = 2;
}
```

### 消息相关

#### CreateMessage

```protobuf
message CreateMessageRequest {
    string conversation_id = 1;
    string from_user_id = 2;
    string to_user_id = 3;
    string content = 4;
    string message_type = 5;
    map<string, string> metadata = 6;
}

message CreateMessageResponse {
    string message_id = 1;
    Message message = 2;
    Error error = 3;
}
```

#### GetMessages

```protobuf
message GetMessagesRequest {
    string conversation_id = 1;
    repeated string message_ids = 2;
    int32 limit = 3;
    int32 offset = 4;
}

message GetMessagesResponse {
    repeated Message messages = 1;
    int32 total = 2;
    Error error = 3;
}
```

#### UpdateMessageStatus

```protobuf
message UpdateMessageStatusRequest {
    string message_id = 1;
    int32 status = 2;
    string user_id = 3;
}

message UpdateMessageStatusResponse {
    bool success = 1;
    Error error = 2;
}
```

#### DeleteMessage

```protobuf
message DeleteMessageRequest {
    string message_id = 1;
    string user_id = 2;
}

message DeleteMessageResponse {
    bool success = 1;
    Error error = 2;
}
```

### 会话相关

#### CreateConversation

```protobuf
message CreateConversationRequest {
    string type = 1;
    string name = 2;
    string avatar = 3;
    repeated string participant_ids = 4;
}

message CreateConversationResponse {
    string conversation_id = 1;
    Conversation conversation = 2;
    Error error = 3;
}
```

#### GetConversation

```protobuf
message GetConversationRequest {
    string conversation_id = 1;
}

message GetConversationResponse {
    Conversation conversation = 1;
    Error error = 2;
}
```

#### UpdateConversation

```protobuf
message UpdateConversationRequest {
    string conversation_id = 1;
    string name = 2;
    string avatar = 3;
}

message UpdateConversationResponse {
    bool success = 1;
    Conversation conversation = 2;
    Error error = 3;
}
```

#### GetConversationsByUser

```protobuf
message GetConversationsByUserRequest {
    string user_id = 1;
    PaginationRequest pagination = 2;
}

message GetConversationsByUserResponse {
    repeated ConversationInfo conversations = 1;
    PaginationResponse pagination = 2;
    Error error = 3;
}
```

#### GetConversationParticipants

```protobuf
message GetConversationParticipantsRequest {
    string conversation_id = 1;
}

message GetConversationParticipantsResponse {
    repeated string participant_ids = 1;
    Error error = 2;
}
```

### 群组相关

#### CreateGroup

```protobuf
message CreateGroupRequest {
    string name = 1;
    string description = 2;
    string avatar = 3;
    string owner_id = 4;
    int32 max_members = 5;
}

message CreateGroupResponse {
    string group_id = 1;
    Group group = 2;
    Error error = 3;
}
```

#### GetGroup

```protobuf
message GetGroupRequest {
    string group_id = 1;
}

message GetGroupResponse {
    Group group = 1;
    Error error = 2;
}
```

#### UpdateGroup

```protobuf
message UpdateGroupRequest {
    string group_id = 1;
    string name = 2;
    string description = 3;
    string avatar = 4;
    int32 max_members = 5;
}

message UpdateGroupResponse {
    bool success = 1;
    Group group = 2;
    Error error = 3;
}
```

#### GetGroupsByUser

```protobuf
message GetGroupsByUserRequest {
    string user_id = 1;
    PaginationRequest pagination = 2;
}

message GetGroupsByUserResponse {
    repeated Group groups = 1;
    PaginationResponse pagination = 2;
    Error error = 3;
}
```

#### AddGroupMember

```protobuf
message AddGroupMemberRequest {
    string group_id = 1;
    string user_id = 2;
    string role = 3;
}

message AddGroupMemberResponse {
    bool success = 1;
    Error error = 2;
}
```

#### RemoveGroupMember

```protobuf
message RemoveGroupMemberRequest {
    string group_id = 1;
    string user_id = 2;
}

message RemoveGroupMemberResponse {
    bool success = 1;
    Error error = 2;
}
```

#### GetGroupMembers

```protobuf
message GetGroupMembersRequest {
    string group_id = 1;
    PaginationRequest pagination = 2;
}

message GetGroupMembersResponse {
    repeated GroupMember members = 1;
    PaginationResponse pagination = 2;
    Error error = 3;
}
```

#### UpdateGroupMemberRole

```protobuf
message UpdateGroupMemberRoleRequest {
    string group_id = 1;
    string user_id = 2;
    string role = 3;
}

message UpdateGroupMemberRoleResponse {
    bool success = 1;
    Error error = 2;
}
```

## 错误处理

### 错误码映射

| gRPC 状态码 | 业务错误码 | 说明 |
|-------------|------------|------|
| OK | ERROR_CODE_NONE | 成功 |
| INVALID_ARGUMENT | ERROR_CODE_INVALID_PARAMS | 参数错误 |
| UNAUTHENTICATED | ERROR_CODE_UNAUTHORIZED | 未认证 |
| PERMISSION_DENIED | ERROR_CODE_PERMISSION_DENIED | 权限不足 |
| NOT_FOUND | ERROR_CODE_NOT_FOUND | 资源不存在 |
| INTERNAL | ERROR_CODE_INTERNAL_ERROR | 内部错误 |
| ALREADY_EXISTS | ERROR_CODE_DUPLICATE_OPERATION | 重复操作 |
| RESOURCE_EXHAUSTED | ERROR_CODE_RATE_LIMIT | 频率限制 |

### 错误响应格式

```protobuf
message ErrorResponse {
    ErrorCode code = 1;
    string message = 2;
    map<string, string> details = 3;
    int64 timestamp = 4;
    string request_id = 5;
}
```

## 性能要求

### 响应时间

- **简单查询**: < 50ms
- **复杂查询**: < 200ms
- **写入操作**: < 100ms
- **批量操作**: < 1000ms

### 并发处理

- **连接池**: 最小 10，最大 100
- **超时时间**: 30s
- **重试机制**: 最多 3 次

## 监控指标

### 服务监控

- **请求量**: QPS
- **响应时间**: P50, P95, P99
- **错误率**: 按错误码统计
- **资源使用**: CPU, 内存, 网络带宽

### 业务监控

- **活跃用户数**: 日活、月活
- **消息量**: 发送量、接收量
- **会话数**: 创建数、活跃数
- **群组数**: 创建数、成员数

## 安全要求

### 认证

- 所有 gRPC 调用必须进行认证
- 使用 TLS 加密传输
- 支持基于 Token 的认证

### 授权

- 基于角色的访问控制
- 用户只能访问自己的数据
- 群组操作需要相应权限

### 数据保护

- 敏感数据脱敏
- 数据传输加密
- 操作日志记录