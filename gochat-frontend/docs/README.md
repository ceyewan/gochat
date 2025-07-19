# GoChat 即时通讯系统

## 项目简介

GoChat 是一个基于 Vue 3 的即时通讯系统前端，支持用户注册登录、游客模式、私聊、群聊和世界聊天室功能。

## 功能特性

### 用户系统
- **用户注册/登录**：支持用户名密码注册登录
- **游客模式**：支持游客快速进入聊天
- **用户信息管理**：头像、昵称等基本信息

### 聊天功能
- **私聊**：注册用户之间的一对一聊天
- **群聊**：多人群组聊天（注册用户）
- **世界聊天室**：所有用户都可参与的公共聊天室

### 实时通信
- **WebSocket 连接**：实时消息推送
- **在线状态**：显示用户在线状态
- **消息确认**：消息发送状态反馈

## 技术栈

- **前端框架**：Vue 3 (Options API)
- **状态管理**：Vuex 4
- **路由管理**：Vue Router 4
- **HTTP客户端**：Axios
- **实时通信**：WebSocket
- **构建工具**：Vite

## 快速开始

### 环境要求
- Node.js >= 16.0.0
- npm >= 8.0.0

### 安装依赖
```bash
npm install
```

### 启动开发服务器
```bash
# 启动前端开发服务器
npm run dev

# 启动 Mock 后端服务器（用于开发测试）
cd mock-server
npm install
npm start
```

### 访问应用
- 前端地址：http://localhost:5173
- Mock API：http://localhost:8080

## 项目结构

```
gochat-frontend/
├── src/                        # 源代码
│   ├── components/             # 组件
│   ├── views/                  # 页面
│   ├── store/                  # 状态管理
│   ├── router/                 # 路由配置
│   └── utils/                  # 工具类
├── mock-server/                # Mock 服务器
├── docs/                       # 文档
│   ├── api_documentation.md    # API 接口文档
│   ├── architecture.md         # 架构文档
│   └── design.md               # 设计文档
└── README.md                   # 项目说明
```

## 开发说明

### 测试账号
Mock 服务器预设了以下测试用户：
- 用户名：张三、李四、王五、赵六
- 密码：任意（Mock 服务器不验证密码）

### 游客模式
- 点击登录页面的"游客登录"按钮
- 系统会自动分配游客昵称
- 游客只能使用世界聊天室功能

### API 接口
详细的 API 接口文档请参考：[API 文档](./docs/api_documentation.md)

### 架构说明
项目架构和技术细节请参考：[架构文档](./docs/architecture.md)

## 构建部署

### 构建生产版本
```bash
npm run build
```

### 预览构建结果
```bash
npm run preview
```

## 下一步开发计划

### 即将添加的功能
1. **游客登录方式**：简化游客进入流程
2. **世界聊天室优化**：改进世界聊天室体验
3. **UI 优化**：提升界面美观度和用户体验

### 技术改进
- 消息分页加载优化
- 移动端适配改进
- 性能优化

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License

---
更新时间：2025-01-19
