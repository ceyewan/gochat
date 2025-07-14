# Mock服务器测试指南

## 服务器启动

1. **启动Mock服务器**：
   ```bash
   cd mock-server
   npm install  # 首次运行需要安装依赖
   npm start    # 启动服务器
   ```

2. **启动前端项目**：
   ```bash
   npm run dev  # 在项目根目录启动前端
   ```

## 测试账号

Mock服务器预设了以下测试用户：

| 用户名 | 密码 | 用户ID | 说明 |
|--------|------|--------|------|
| 张三   | 任意 | user1  | 测试用户1 |
| 李四   | 任意 | user2  | 测试用户2 |
| 王五   | 任意 | user3  | 测试用户3 |
| 赵六   | 任意 | user4  | 测试用户4 |

> **注意**：Mock服务器对密码验证很宽松，任何密码都可以登录现有用户

## 功能测试

### 1. 用户认证测试

#### 登录功能
1. 访问 `http://localhost:5173`
2. 使用测试账号登录（如：用户名`张三`，密码任意）
3. 登录成功后会跳转到聊天界面

#### 注册功能
1. 点击"没有账号？立即注册"
2. 输入新用户名（至少3个字符）和密码（至少6个字符）
3. 注册成功后自动跳转到登录页

### 2. 聊天功能测试

#### 会话列表
- 登录后左侧显示会话列表
- 显示未读消息数量
- 点击会话项切换到对应聊天

#### 发送消息
1. 选择一个会话
2. 在底部输入框输入消息
3. 按回车或点击发送按钮
4. 消息会显示在聊天区域

#### 实时通信
1. 打开两个浏览器标签页
2. 分别登录不同用户（如张三和李四）
3. 在一个标签页发送消息
4. 另一个标签页应该能实时接收到消息

### 3. 好友管理测试

#### 添加好友
1. 点击会话列表底部的"添加好友"按钮
2. 输入现有用户名（如"李四"）进行搜索
3. 点击"添加好友"按钮
4. 添加成功后会自动创建会话

#### 搜索用户
- 在添加好友弹窗中搜索不存在的用户会显示"用户不存在"
- 搜索现有用户会显示用户信息卡片

### 4. 群聊功能测试

#### 创建群聊
1. 点击会话列表底部的"创建群聊"按钮
2. 输入群名称
3. 选择要邀请的成员
4. 点击"创建群聊"按钮
5. 创建成功后会出现在会话列表中

### 5. 在线状态测试

- 单聊会话中显示对方在线状态
- 群聊显示成员数量
- WebSocket连接状态在控制台可见

## API接口测试

可以使用Postman或curl测试REST API：

### 认证接口
```bash
# 登录
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "张三", "password": "123456"}'

# 注册
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "测试用户", "password": "123456"}'
```

### 会话接口
```bash
# 获取会话列表 (需要token)
curl -X GET http://localhost:8080/api/conversations \
  -H "Authorization: Bearer mock_token_user1_xxxx"

# 获取消息历史
curl -X GET "http://localhost:8080/api/conversations/conv1/messages?page=1&size=20" \
  -H "Authorization: Bearer mock_token_user1_xxxx"
```

## WebSocket测试

可以使用浏览器开发者工具或WebSocket客户端测试：

1. 连接: `ws://localhost:8080/ws?token=mock_token_user1_xxxx`
2. 发送消息:
   ```json
   {
     "type": "send-message",
     "data": {
       "conversationId": "conv1",
       "content": "Hello World",
       "messageType": "text"
     }
   }
   ```

## 故障排除

### 常见问题

1. **WebSocket连接失败**
   - 检查Mock服务器是否运行
   - 确认WebSocket URL配置正确
   - 查看浏览器控制台错误信息

2. **API请求失败**
   - 检查网络代理配置
   - 确认后端服务器运行在8080端口
   - 查看Mock服务器控制台日志

3. **页面跳转失败**
   - 检查路由配置
   - 确认token是否正确保存
   - 查看Vue Router错误信息

### 开发者工具

1. **浏览器控制台**：查看前端错误和WebSocket消息
2. **网络面板**：查看API请求和响应
3. **Mock服务器日志**：查看后端请求处理过程

## 预设数据

Mock服务器包含以下预设数据：

- **用户**：4个测试用户
- **会话**：3个预设会话（2个单聊，1个群聊）
- **消息**：每个会话包含历史消息
- **好友关系**：用户间的好友关系
- **群聊**：1个技术交流群

所有数据在服务器重启后会重置到初始状态。

## 扩展功能

可以根据需要扩展Mock服务器：

1. **添加新的API接口**
2. **修改模拟数据**
3. **增加WebSocket消息类型**
4. **实现数据持久化**
5. **添加文件上传功能**

---

更新时间：2025-01-13 23:14
