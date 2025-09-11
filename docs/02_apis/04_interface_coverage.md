# GoChat 接口覆盖清单

本文档提供 GoChat 系统所有接口的完整清单，确保没有遗漏重要功能。

## 1. HTTP API 接口清单

### 1.1 认证相关 (`/auth`)
| 接口 | 方法 | 描述 | 文档覆盖 | 实现状态 |
|------|------|------|----------|----------|
| `/auth/register` | POST | 用户注册 | ✅ | ✅ |
| `/auth/login` | POST | 用户登录 | ✅ | ✅ |
| `/auth/guest` | POST | 游客登录 | ✅ | ✅ |
| `/auth/logout` | POST | 用户登出 | ✅ | ✅ |
| `/auth/refresh` | POST | 刷新令牌 | ✅ | 📋 |

### 1.2 用户相关 (`/users`)
| 接口 | 方法 | 描述 | 文档覆盖 | 实现状态 |
|------|------|------|----------|----------|
| `/users/profile` | GET | 获取用户资料 | ✅ | ✅ |
| `/users/profile` | PUT | 更新用户资料 | ✅ | 📋 |
| `/users/search` | GET | 搜索用户 | ✅ | 📋 |

### 1.3 好友管理 (`/friends`)
| 接口 | 方法 | 描述 | 文档覆盖 | 实现状态 |
|------|------|------|----------|----------|
| `/friends` | GET | 获取好友列表 | ✅ | 📋 |
| `/friends/requests` | GET | 获取好友申请列表 | ✅ | 📋 |
| `/friends/requests` | POST | 发送好友申请 | ✅ | 📋 |
| `/friends/requests/{id}` | PUT | 处理好友申请 | ✅ | 📋 |

### 1.4 会话管理 (`/conversations`)
| 接口 | 方法 | 描述 | 文档覆盖 | 实现状态 |
|------|------|------|----------|----------|
| `/conversations` | GET | 获取会话列表 | ✅ | ✅ |
| `/conversations` | POST | 创建会话 | ✅ | ✅ |
| `/conversations/{id}` | GET | 获取会话详情 | ✅ | ✅ |
| `/conversations/{id}` | PUT | 更新会话信息 | ✅ | 📋 |
| `/conversations/{id}` | DELETE | 删除会话 | ⚠️ | 📋 |
| `/conversations/{id}/messages` | GET | 获取消息历史 | ✅ | ✅ |
| `/conversations/{id}/messages` | POST | 发送消息 | ✅ | ✅ |
| `/conversations/{id}/read` | PUT | 标记已读 | ✅ | ✅ |

### 1.5 会话成员管理 (`/conversations/{id}/members`)
| 接口 | 方法 | 描述 | 文档覆盖 | 实现状态 |
|------|------|------|----------|----------|
| `/conversations/{id}/members` | GET | 获取成员列表 | ✅ | ✅ |
| `/conversations/{id}/members` | POST | 添加成员 | ✅ | ✅ |
| `/conversations/{id}/members/{userId}` | DELETE | 移除成员 | ✅ | ✅ |
| `/conversations/{id}/members/{userId}` | PUT | 更新成员角色 | ✅ | ✅ |

## 2. WebSocket 消息类型清单

### 2.1 客户端 → 服务器消息
| 消息类型 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `send-message` | 发送消息 | ✅ | ✅ |
| `mark-read` | 标记已读 | ✅ | ✅ |
| `ping` | 心跳保活 | ✅ | ✅ |
| `typing` | 正在输入 | ❌ | 📋 |
| `stop-typing` | 停止输入 | ❌ | 📋 |

### 2.2 服务器 → 客户端消息
| 消息类型 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `new-message` | 新消息通知 | ✅ | ✅ |
| `message-ack` | 消息确认 | ✅ | ✅ |
| `online-status` | 在线状态更新 | ✅ | ✅ |
| `pong` | 心跳响应 | ✅ | ✅ |
| `error` | 错误通知 | ✅ | ✅ |
| `conversation-updated` | 会话更新 | ❌ | 📋 |
| `member-added` | 成员添加 | ❌ | 📋 |
| `member-removed` | 成员移除 | ❌ | 📋 |
| `typing-indicator` | 输入状态 | ❌ | 📋 |

## 3. gRPC 接口清单

### 3.1 im-logic 服务接口

#### AuthService
| RPC 方法 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `Login` | 用户登录 | ✅ | ✅ |
| `Register` | 用户注册 | ✅ | ✅ |
| `GuestLogin` | 游客登录 | ✅ | ✅ |
| `RefreshToken` | 刷新令牌 | ✅ | 📋 |
| `Logout` | 用户登出 | ✅ | 📋 |
| `ValidateToken` | 验证令牌 | ✅ | ✅ |

#### ConversationService
| RPC 方法 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `CreateConversation` | 创建会话 | ✅ | ✅ |
| `GetConversation` | 获取会话详情 | ✅ | ✅ |
| `GetConversations` | 获取会话列表 | ✅ | ✅ |
| `GetConversationsOptimized` | 优化会话列表查询 | ✅ | 📋 |
| `UpdateConversation` | 更新会话信息 | ⚠️ | 📋 |
| `DeleteConversation` | 删除会话 | ⚠️ | 📋 |
| `AddMembers` | 添加成员 | ✅ | ✅ |
| `RemoveMembers` | 移除成员 | ✅ | ✅ |
| `UpdateMemberRole` | 更新成员角色 | ✅ | ✅ |
| `GetMembers` | 获取成员列表 | ✅ | ✅ |
| `LeaveConversation` | 离开会话 | ⚠️ | 📋 |
| `GetMessages` | 获取消息历史 | ✅ | ✅ |
| `MarkAsRead` | 标记已读 | ✅ | ✅ |
| `GetUnreadCount` | 获取未读数 | ✅ | ✅ |
| `JoinWorldChat` | 加入世界聊天室 | ⚠️ | 📋 |
| `SearchConversations` | 搜索会话 | ⚠️ | 📋 |

#### MessageService
| RPC 方法 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `SendMessage` | 发送消息 | ✅ | ✅ |

#### FriendService (缺失)
| RPC 方法 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `SendFriendRequest` | 发送好友申请 | ✅ | ❌ |
| `HandleFriendRequest` | 处理好友申请 | ✅ | ❌ |
| `GetFriends` | 获取好友列表 | ✅ | ❌ |
| `GetFriendRequests` | 获取好友申请 | ✅ | ❌ |
| `SearchUsers` | 搜索用户 | ✅ | ❌ |

### 3.2 im-repo 服务接口

#### UserService
| RPC 方法 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `CreateUser` | 创建用户 | ✅ | ✅ |
| `GetUser` | 获取用户信息 | ✅ | ✅ |
| `GetUsers` | 批量获取用户 | ✅ | ✅ |
| `UpdateUser` | 更新用户信息 | ✅ | 📋 |
| `VerifyPassword` | 验证密码 | ✅ | ✅ |
| `GetUserByUsername` | 根据用户名获取用户 | ✅ | ✅ |
| `SearchUsersByUsername` | 搜索用户 | ✅ | ❌ |

#### ConversationService
| RPC 方法 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `GetUserConversations` | 获取用户会话列表 | ✅ | ✅ |
| `GetUserConversationsWithDetails` | 获取详细会话列表 | ✅ | 📋 |
| `UpdateReadPointer` | 更新已读位置 | ✅ | ✅ |
| `GetUnreadCount` | 获取未读数 | ✅ | ✅ |
| `GetReadPointer` | 获取已读位置 | ✅ | ✅ |
| `BatchGetUnreadCounts` | 批量获取未读数 | ✅ | 📋 |
| `CreateConversation` | 创建会话 | ✅ | ✅ |
| `BatchGetConversations` | 批量获取会话信息 | ✅ | 📋 |
| `AddConversationMember` | 添加会话成员 | ✅ | ✅ |
| `RemoveConversationMember` | 移除会话成员 | ✅ | ✅ |
| `UpdateMemberRole` | 更新成员角色 | ✅ | ✅ |
| `GetConversationMembers` | 获取会话成员 | ✅ | ✅ |

#### MessageService
| RPC 方法 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `SaveMessage` | 保存消息 | ✅ | ✅ |
| `GetMessage` | 获取单条消息 | ⚠️ | 📋 |
| `GetConversationMessages` | 获取会话消息列表 | ✅ | ✅ |
| `CheckMessageIdempotency` | 检查消息幂等性 | ✅ | 📋 |
| `GetLatestMessages` | 获取最新消息 | ⚠️ | 📋 |
| `DeleteMessage` | 删除消息 | ✅ | 📋 |

#### OnlineStatusService
| RPC 方法 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `SetUserOnline` | 设置用户在线 | ✅ | ✅ |
| `SetUserOffline` | 设置用户离线 | ✅ | ✅ |
| `GetUserOnlineStatus` | 获取在线状态 | ✅ | ✅ |
| `GetUsersOnlineStatus` | 批量获取在线状态 | ✅ | ✅ |
| `UpdateHeartbeat` | 更新心跳 | ✅ | 📋 |
| `CleanupExpiredStatus` | 清理过期状态 | ✅ | 📋 |

#### FriendService (缺失)
| RPC 方法 | 描述 | 文档覆盖 | 实现状态 |
|----------|------|----------|----------|
| `CreateFriendRequest` | 创建好友申请 | ✅ | ❌ |
| `UpdateFriendRequest` | 更新申请状态 | ✅ | ❌ |
| `GetUserFriends` | 获取用户好友 | ✅ | ❌ |
| `GetUserFriendRequests` | 获取好友申请 | ✅ | ❌ |

## 4. Kafka 消息类型清单

### 4.1 核心消息流
| Topic | 生产者 | 消费者 | 描述 | 文档覆盖 |
|-------|--------|--------|------|----------|
| `gochat.messages.upstream` | im-gateway | im-logic | 上行消息 | ✅ |
| `gochat.messages.persist` | im-logic | im-task | 持久化消息 | ✅ |
| `gochat.messages.downstream.{id}` | im-logic/im-task | im-gateway | 下行消息 | ✅ |
| `gochat.tasks.fanout` | im-logic | im-task | 扇出任务 | ✅ |

### 4.2 领域事件
| Topic | 描述 | 文档覆盖 | 使用状态 |
|-------|------|----------|----------|
| `gochat.user-events` | 用户事件 | ✅ | ✅ |
| `gochat.conversation-events` | 会话事件 | ✅ | ✅ |
| `gochat.message-events` | 消息事件 | ✅ | ✅ |
| `gochat.friend-events` | 好友事件 | ✅ | 📋 |
| `gochat.system-notifications` | 系统通知 | ✅ | 📋 |

## 5. 遗漏功能清单

### 5.1 高优先级遗漏 (需要补充)
- ❌ **好友管理完整实现**: proto定义、RPC实现、HTTP接口
- ❌ **用户搜索功能**: 搜索接口实现和优化
- ❌ **Token刷新机制**: 自动刷新和黑名单
- ❌ **消息撤回功能**: 消息删除和撤回通知
- ❌ **输入状态指示**: typing indicator功能

### 5.2 中优先级遗漏 (后续补充)
- ⚠️ **会话设置管理**: 群聊设置、通知设置
- ⚠️ **文件消息支持**: 图片、文件上传下载
- ⚠️ **消息搜索功能**: 全文搜索和过滤
- ⚠️ **用户黑名单**: 拉黑和屏蔽功能

### 5.3 低优先级功能 (长期规划)
- 📋 **消息转发**: 消息转发到其他会话
- 📋 **群公告管理**: 群公告发布和管理
- 📋 **会话置顶**: 重要会话置顶显示
- 📋 **消息引用**: 回复特定消息

## 6. 接口一致性检查

### 6.1 数据模型一致性 ✅
- 统一使用 `common/v1/types.proto` 中的类型定义
- 消除了不同层次间的类型不匹配问题

### 6.2 错误处理一致性 ✅
- 统一的错误码定义
- 标准化的错误响应格式

### 6.3 分页接口一致性 ✅
- HTTP API 使用 page/pageSize 分页
- gRPC 使用 offset/limit 分页
- 游标分页用于高性能场景

## 7. 总结

**当前覆盖情况：**
- ✅ **完整覆盖**: 核心消息收发、会话管理、用户认证
- ⚠️ **部分覆盖**: 好友管理、用户搜索、消息增强功能  
- ❌ **缺失功能**: 输入状态、文件消息、消息搜索

**下一步工作：**
1. 补充好友管理的完整 proto 定义和实现
2. 实现用户搜索和资料管理功能
3. 添加消息撤回和输入状态功能
4. 完善错误处理和监控指标

通过这个清单可以确保 GoChat 系统功能的完整性和一致性。