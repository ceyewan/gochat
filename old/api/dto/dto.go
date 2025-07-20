package dto

import "net/http"

// 定义 Code 常量
const (
	CodeSuccess = 200
	CodeFail    = 400
)

// 定义通用响应状态码
const (
	StatusOK          = http.StatusOK
	StatusBadRequest  = http.StatusBadRequest
	StatusUnauth      = http.StatusUnauthorized
	StatusServerError = http.StatusInternalServerError
)

// 用户认证相关请求和响应结构体定义
type (
	// AuthRequest 用户认证请求结构体，用于注册和登录请求
	AuthRequest struct {
		Username string `json:"username" binding:"required"` // 用户名，必填字段
		Password string `json:"password" binding:"required"` // 密码，必填字段
	}

	// LoginResponse 登录成功响应数据
	LoginResponse struct {
		Code     int    `json:"code"`    // 业务状态码
		UserID   int    `json:"user_id"` // 用户ID
		UserName string `json:"username"`
		Token    string `json:"token"` // 身份验证令牌
	}

	// RegisterResponse 注册成功响应数据
	RegisterResponse struct {
		Code int `json:"code"` // 业务状态码
	}

	// TokenRequest 令牌请求结构体，用于注销和身份验证请求
	TokenRequest struct {
		Token string `json:"token" binding:"required"` // 认证令牌，必填字段
	}

	// LogoutResponse 注销成功响应数据
	LogoutResponse struct {
		Code int `json:"code"` // 业务状态码
	}

	// CheckAuthResponse 身份验证响应数据
	CheckAuthResponse struct {
		Code int `json:"code"` // 业务状态码
	}

	// ErrorResponse 错误响应数据
	ErrorResponse struct {
		Code  int    `json:"code"`  // 业务状态码
		Error string `json:"error"` // 错误信息
	}

	// PushRequest 消息推送请求结构体
	PushRequest struct {
		Msg          string `json:"msg" binding:"required"`            // 消息内容，必填字段
		ToUserID     int    `json:"to_user_id" binding:"required"`     // 接收用户ID，必填字段
		ToUserName   string `json:"to_user_name" binding:"required"`   // 接收用户名，必填字段
		FromUserID   int    `json:"from_user_id" binding:"required"`   // 发送用户ID，必填字段
		FromUserName string `json:"from_user_name" binding:"required"` // 发送用户名，必填字段
		RoomID       int    `json:"room_id" binding:"required"`        // 房间ID，必填字段
		Token        string `json:"token" binding:"required"`          // 认证令牌，必填字段
	}

	// PushResponse 消息推送响应数据
	PushResponse struct {
		Code int `json:"code"` // 业务状态码
	}
)
