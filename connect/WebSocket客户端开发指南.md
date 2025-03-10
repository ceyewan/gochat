# WebSocket客户端开发指南

## 概述

本文档描述了如何使用WebSocket协议与GoChat服务进行实时通信。WebSocket接口用于实现即时消息的发送和接收。

## 连接建立

1. 建立WebSocket连接：
   - URL: `ws://{host}/ws`
   - 示例：`ws://localhost:8080/ws`

2. 连接成功后，需要发送认证消息

    token 在登录后获取，userID 在获取 token 后调用 checktoken 获取，roomID 是用户指定要加入的房间，默认为 0，表示 gochat 房间。

   ```json
   {
     "user_id": 0,
     "room_id": 0,
     "token": "your_auth_token",
     "message": "connect"
   }
   ```

3. 认证成功后，后续接收消息推送即可

## 消息格式

### 客户端消息格式

```json
{
  "user_id": 123,
  "room_id": 456,
  "token": "string",
  "message": "string"
}
```

### 服务端消息格式

```json
{
  "count": 1,
  "msg": "string",
  "room_user_info": {
    "user1": "info1",
    "user2": "info2"
  }
}
```

如果 count 不为 -1，则更新 count；如果 msg 不为 ""，则显示消息；如果 room_user_info 不为 nil，则更新在线用户。

## 消息发送与接收

### 发送消息

1. 构造消息体：

   ```json
   {
     "user_id": 123,
     "room_id": 456,
     "token": "your_auth_token",
     "message": "Hello, World!"
   }
   ```

2. 通过WebSocket连接发送消息

### 接收消息

1. 监听WebSocket的message事件
2. 解析接收到的消息
3. 根据消息类型进行相应处理

## 示例代码

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  // 发送认证消息
  ws.send(JSON.stringify({
    user_id: 0,
    room_id: 0,
    token: 'your_auth_token',
    message: 'connect'
  }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log('Received:', msg);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket connection closed');
};
```
