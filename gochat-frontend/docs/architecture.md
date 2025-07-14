# 项目架构文档

## 项目概览

分布式微服务即时通讯系统前端项目，采用Vue 3 + Vuex 4 + Vue Router 4技术栈，配合Mock服务器进行开发和测试。

## 技术栈

- **前端框架**：Vue 3 (Options API)
- **状态管理**：Vuex 4
- **路由管理**：Vue Router 4
- **HTTP客户端**：Axios
- **实时通信**：WebSocket (原生API + 自定义封装)
- **构建工具**：Vite
- **样式方案**：原生CSS + Flexbox布局
- **Mock服务器**：Express.js + ws

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
├── mock-server/               # Mock服务器
│   ├── server.js              # 服务器主文件
│   ├── package.json           # 依赖配置
│   ├── data/                  # 模拟数据
│   │   └── mockData.js        # 数据定义
│   └── routes/                # API路由
│       ├── auth.js            # 认证相关API
│       ├── users.js           # 用户管理API
│       ├── conversations.js   # 会话管理API
│       ├── messages.js        # 消息管理API
│       ├── friends.js         # 好友管理API
│       └── groups.js          # 群聊管理API
├── docs/                      # 项目文档
│   ├── design.md              # 设计文档
│   ├── plan.md                # 开发计划
│   ├── dev_log.md             # 开发日志
│   ├── issues_and_solutions.md # 问题记录
│   ├── mock_server_guide.md   # Mock服务器使用指南
│   └── architecture.md        # 本架构文档
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
**职责**：用户和群成员在线状态管理

**核心状态**：
```javascript
state: {
    friendsStatus: {},      // 好友在线状态
    groupMembersStatus: {}  // 群成员在线状态
}
```

### 3. 工具类模块 (src/utils/)

#### 3.1 HTTP请求封装 (request.js)
**功能**：统一的API请求处理

**特性**：
- 自动添加Authorization header
- 统一错误处理（401自动登出）
- 请求/响应拦截器
- 超时和重试机制

#### 3.2 WebSocket封装 (websocket.js)
**功能**：实时通信连接管理

**特性**：
- 自动重连机制（最多3次）
- 心跳检测
- 消息类型分发
- 连接状态管理

**消息类型**：
- `connected` - 连接确认
- `new-message` - 新消息
- `message-ack` - 消息确认
- `friend-online` - 好友上线
- `group-member-online` - 群成员上线

### 4. 组件模块 (src/components/)

#### 4.1 页面级组件 (src/views/)
- **Login.vue** - 登录页面
- **Register.vue** - 注册页面
- **ChatLayout.vue** - 聊天主界面布局

#### 4.2 功能组件
- **Header.vue** - 顶部导航栏（用户信息、登出）
- **ConversationList.vue** - 会话列表容器
- **ConversationItem.vue** - 单个会话项
- **ChatMain.vue** - 聊天主区域
- **MessageBubble.vue** - 消息气泡

#### 4.3 弹窗组件
- **AddFriendModal.vue** - 添加好友弹窗
- **CreateGroupModal.vue** - 创建群聊弹窗

## 数据流设计

### 1. 用户认证流程
```
Login.vue → user/login → 保存token → 跳转/chat → 初始化WebSocket
```

### 2. 消息发送流程
```
ChatMain.vue → currentChat/sendMessage → WebSocket发送 → 服务器确认 → 更新消息状态
```

### 3. 消息接收流程
```
WebSocket接收 → currentChat/receiveMessage → 更新消息列表 → UI更新
```

### 4. 会话管理流程
```
ConversationList.vue → conversations/fetchConversations → 显示会话列表
点击会话 → currentChat/selectConversation → 加载消息历史
```

## 开发规范

### 1. 组件命名
- 页面组件：PascalCase (如：ChatLayout.vue)
- 功能组件：PascalCase (如：MessageBubble.vue)
- 通用组件：放在common/目录下

### 2. 状态管理
- 模块化：按功能拆分状态模块
- 命名空间：使用`namespaced: true`
- 异步操作：统一使用actions处理

### 3. 样式规范
- 使用scoped样式避免污染
- 响应式设计：支持移动端适配
- CSS变量：定义主题色彩

### 4. 错误处理
- HTTP错误：统一在request.js拦截器处理
- WebSocket错误：在websocket.js中处理重连
- 组件错误：使用try-catch包装异步操作

## 性能优化

### 1. 代码分割
- 路由懒加载
- 组件动态导入
- 第三方库按需引入

### 2. 缓存策略
- 用户信息缓存在localStorage
- 会话列表缓存在Vuex
- 图片懒加载和缓存

### 3. 网络优化
- HTTP请求合并
- WebSocket连接复用
- 消息分页加载

## 扩展接口

### 1. 插件系统
- 支持自定义消息类型
- 支持扩展UI组件
- 支持第三方集成

### 2. 主题系统
- CSS变量定义主题
- 动态切换主题色
- 支持暗黑模式

### 3. 国际化
- Vue I18n集成准备
- 多语言支持预留
- 日期时间本地化

---
更新时间：2025-01-13 23:58
