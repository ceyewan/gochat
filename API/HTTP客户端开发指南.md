# GoChat API 文档

## 概述

本文档描述了GoChat即时通讯服务的API接口，包括用户认证、消息推送等功能。所有API均采用RESTful风格，请求和响应数据格式为JSON。

## 认证API

### 用户登录

- 请求路径: `/api/login`
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

- 请求路径: `/api/register`
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

- 请求路径: `/api/logout`
- 请求方法: POST
- 请求参数:

  ```json
  {
    "token": "string"
  }
  ```

- 响应示例:

  ```json
  {
    "code": 200
  }
  ```

### 认证状态检查

- 请求路径: `/api/checkAuth`
- 请求方法: POST
- 请求参数:

  ```json
  {
    "token": "string"
  }
  ```

- 响应示例:

  ```json
  {
    "code": 200,
    "data": {
      "userid": 123,
      "username": "string"
    }
  }
  ```

## 消息推送API

### 单聊消息推送

- 请求路径: `/api/push`
- 请求方法: POST
- 请求参数:

  ```json
  {
    "msg": "string",
    "toUserId": 123,
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

### 群聊消息推送

- 请求路径: `/api/pushRoom`
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
| 500    | 服务器内部错误 |
