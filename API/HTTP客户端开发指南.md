# GoChat API 文档

## 概述

本文档描述了GoChat即时通讯服务的API接口，包括用户认证、消息推送等功能。所有API均采用RESTful风格，请求和响应数据格式为JSON。

## 认证API

### 用户登录

- 请求路径: `/user/login`
- 请求方法: POST
- 请求参数:

  ```json
  {
    "username": "string",
    "password": "string"
  }
  ```

- 响应示例:

  ```json
  {
    "code": 200,
    "data": {
      "token": "string"
    }
  }
  ```

### 用户注册

- 请求路径: `/user/register`
- 请求方法: POST
- 请求参数:

  ```json
  {
    "username": "string",
    "password": "string"
  }
  ```

- 响应示例:

  ```json
  {
    "code": 200,
    "data": {
      "username": "string"
    }
  }
  ```

### 用户登出

- 请求路径: `/user/logout`
- 请求方法: POST
- 请求参数:

  ```json
  {
    "authToken": "string"
  }
  ```

- 响应示例:

  ```json
  {
    "code": 200
  }
  ```

### 认证状态检查

- 请求路径: `/user/checkAuth`
- 请求方法: POST
- 请求参数:

  ```json
  {
    "authToken": "string"
  }
  ```

- 响应示例:

  ```json
  {
    "code": 200,
    "data": {
      "userId": 123,
      "username": "string"
    }
  }
  ```

## 消息推送API

### 单聊消息推送

- 请求路径: `/push/push`
- 请求方法: POST
- 请求参数:

  ```json
  {
    "msg": "string",
    "toUserId": 123,
    "authToken": "string"
  }
  ```

- 响应示例:

  ```json
  {
    "code": 200
  }
  ```

### 群聊消息推送

- 请求路径: `/push/pushRoom`
- 请求方法: POST
- 请求参数:

  ```json
  {
    "msg": "string",
    "roomId": 456,
    "authToken": "string"
  }
  ```

- 响应示例:

  ```json
  {
    "code": 200
  }
  ```

## 会话验证

所有需要认证的API请求必须在请求体中包含authToken字段：

```json
{
  "authToken": "string"
}
```

会话验证中间件会：

1. 验证authToken的有效性
2. 将用户信息（userId, userName）添加到请求上下文
3. 如果验证失败，返回401状态码

## 跨域支持

所有API均支持CORS跨域请求，响应头包含：

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
Access-Control-Allow-Methods: GET, OPTIONS, POST, PUT, DELETE
```

## 404处理

对于不存在的API路径，系统会返回404状态码，响应格式：

```json
{
  "code": 404,
  "error": "404 Not Found"
}
```

## 响应格式

所有API响应均采用以下格式：

```json
{
  "code": "int",    // 状态码
  "error": "string", // 错误信息
  "data": {}        // 响应数据
}
```

## 错误码

| 状态码 | 描述 |
| ------ | ---- |
| 200    | 请求成功 |
| 400    | 无效的请求参数 |
| 401    | 认证失败 |
| 404    | 资源未找到 |
| 500    | 服务器内部错误 |
