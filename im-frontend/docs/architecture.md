# GoChat 前端架构文档

## 项目概览

GoChat 即时通讯系统前端，基于 Vue 3 技术栈开发，支持用户注册登录、游客模式、私聊、群聊和世界聊天室功能。

## 技术栈

- **前端框架**：Vue 3 (Options API)
- **状态管理**：Vuex 4
- **路由管理**：Vue Router 4
- **HTTP客户端**：Axios
- **实时通信**：WebSocket
- **构建工具**：Vite
- **样式方案**：原生CSS + Flexbox布局

## 项目目录结构

```
gochat-frontend/
├── public/                     # 静态资源
├── src/                        # 源代码目录
│   ├── main.js                # 应用入口文件
│   ├── App.vue                # 根组件
│   ├── style.css              # 全局样式
│   ├── assets/                # 静态资源
│   │   ├── default-avatar.png # 默认头像
│   │   └── vue.svg            # Vue Logo
│   ├── components/            # 组件目录
│   │   ├── common/            # 通用组件
│   │   │   └── Header.vue     # 顶部导航栏
│   │   ├── ConversationList.vue     # 会话列表
│   │   ├── ConversationItem.vue     # 会话列表项
│   │   ├── ChatMain.vue             # 聊天主区域
│   │   ├── MessageBubble.vue        # 消息气泡
│   │   ├── AddFriendModal.vue       # 添加好友弹窗
│   │   └── CreateGroupModal.vue     # 创建群聊弹窗
│   ├── views/                 # 页面组件
│   │   ├── Login.vue          # 登录页
│   │   ├── Register.vue       # 注册页
│   │   └── ChatLayout.vue     # 聊天主界面布局
│   ├── router/                # 路由配置
│   │   └── index.js           # 路由定义和守卫
│   ├── store/                 # 状态管理
│   │   ├── index.js           # Vuex store入口
│   │   └── modules/           # 状态模块
│   │       ├── user.js        # 用户状态模块
│   │       ├── conversations.js    # 会话管理模块
│   │       ├── currentChat.js      # 当前聊天模块
│   │       └── onlineStatus.js     # 在线状态模块
│   └── utils/                 # 工具类
│       ├── request.js         # HTTP请求封装
│       └── websocket.js       # WebSocket连接封装
├── mock-server/               # Mock服务器（开发测试用）
│   ├── server.js              # 服务器主文件
│   ├── package.json           # 依赖配置
│   ├── data/                  # 模拟数据
│   └── routes/                # API路由
├── docs/                      # 项目文档
│   ├── api_documentation.md   # API接口文档
│   ├── architecture.md        # 架构文档
│   └── design.md              # 设计文档
├── .env                       # 环境变量配置
├── vite.config.js            # Vite配置
├── package.json              # 项目依赖
└── README.md                 # 项目说明
```

## 核心模块详解

### 1. 路由模块 (src/router/index.js)

**功能**：页面路由管理和访问控制

**核心特性**：
- 路由守卫：未登录用户自动跳转到登录页
- 动态路由：根据用户状态决定可访问的页面
- 路由懒加载：提升应用性能

**主要路由**：
- `/login` - 登录页
- `/register` - 注册页  
- `/chat` - 聊天主界面（需要认证）

### 2. 状态管理模块 (src/store/)

#### 2.1 用户模块 (user.js)
**职责**：用户认证和信息管理

**核心状态**：
```javascript
state: {
    userInfo: null,     // 用户信息
    token: null,        // 认证令牌
    isLoggedIn: false   // 登录状态
}
```

**核心方法**：
- `login()` - 用户登录
- `register()` - 用户注册
- `logout()` - 用户登出
- `fetchUserInfo()` - 获取用户信息

#### 2.2 会话管理模块 (conversations.js)
**职责**：会话列表管理和好友群聊操作

**核心状态**：
```javascript
state: {
    conversations: [],  // 会话列表
    loading: false      // 加载状态
}
```

**核心方法**：
- `fetchConversations()` - 获取会话列表
- `addFriend()` - 添加好友
- `createGroup()` - 创建群聊
- `markAsRead()` - 标记已读

#### 2.3 当前聊天模块 (currentChat.js)
**职责**：当前会话的消息管理

**核心状态**：
```javascript
state: {
    currentConversation: null,  // 当前会话
    messages: [],              // 消息列表
    loading: false,            // 加载状态
    hasMore: true              // 是否有更多消息
}
```

**核心方法**：
- `selectConversation()` - 选择会话
- `fetchMessages()` - 获取消息历史
- `sendMessage()` - 发送消息
- `receiveMessage()` - 接收消息

#### 2.4 在线状态模块 (onlineStatus.js)
**职责**：用户在线状态管理

**核心状态**：
```javascript
state: {
    friendsStatus: {}      // 好友在线状态
}
```

### 3. 工具类模块 (src/utils/)

#### 3.1 HTTP请求封装 (request.js)
- 自动添加Authorization header
- 统一错误处理（401自动登出）
- 请求/响应拦截器

#### 3.2 WebSocket封装 (websocket.js)
- 自动重连机制
- 心跳检测
- 消息类型分发

### 4. 组件模块

#### 页面组件 (src/views/)
- **Login.vue** - 登录页面
- **Register.vue** - 注册页面
- **ChatLayout.vue** - 聊天主界面

#### 功能组件 (src/components/)
- **Header.vue** - 顶部导航栏
- **ConversationList.vue** - 会话列表
- **ConversationItem.vue** - 会话项
- **ChatMain.vue** - 聊天区域
- **MessageBubble.vue** - 消息气泡
- **AddFriendModal.vue** - 添加好友弹窗
- **CreateGroupModal.vue** - 创建群聊弹窗

## 核心功能

### 1. 用户系统
- 用户注册/登录
- 游客模式登录
- 用户信息管理

### 2. 聊天功能
- 私聊（注册用户）
- 群聊（注册用户）
- 世界聊天室（所有用户）

### 3. 实时通信
- WebSocket 连接管理
- 消息实时推送
- 在线状态同步

---
更新时间：2025-01-19
