# GoChat 即时通讯系统

基于 Vue 3 的即时通讯系统前端，支持用户注册登录、游客模式、私聊、群聊和世界聊天室功能。

## 功能特性

- **用户系统**：注册/登录、游客模式
- **聊天功能**：私聊、群聊、世界聊天室
- **实时通信**：WebSocket 实时消息推送
- **响应式设计**：支持桌面端和移动端

## 快速开始

### 安装依赖
```bash
npm install
```

### 启动开发服务器
```bash
# 启动前端
npm run dev

# 启动 Mock 后端（另开终端）
cd mock-server
npm install
npm start
```

### 访问应用
- 前端：http://localhost:5173
- Mock API：http://localhost:8080

## 测试账号

- 用户名：张三、李四、王五、赵六
- 密码：任意（Mock 服务器不验证密码）
- 或点击"游客登录"快速体验

## 项目文档

- [API 接口文档](./docs/api_documentation.md)
- [架构文档](./docs/architecture.md)
- [设计方案](./docs/design.md)
- [开发指南](./docs/README.md)

## 技术栈

- Vue 3 + Vuex 4 + Vue Router 4
- Axios + WebSocket
- Vite + 原生 CSS

## 下一步开发

1. 游客登录优化
2. 世界聊天室功能完善
3. UI 界面美化

---
更新时间：2025-01-19
