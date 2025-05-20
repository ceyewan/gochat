package handler

import (
	"gochat/api/dto"
	"gochat/api/rpc"
	"gochat/clog"

	"github.com/gin-gonic/gin"
)

// 处理请求参数绑定的通用函数
func bindJSON(c *gin.Context, req interface{}, action string) bool {
	if err := c.BindJSON(req); err != nil {
		clog.Module("user").Warnf("Failed to parse %s request: %v", action, err)
		c.JSON(dto.StatusBadRequest, dto.ErrorResponse{
			Code:  dto.CodeFail,
			Error: "请求参数错误，请检查后重试",
		})
		return false
	}
	return true
}

// Login 处理用户登录请求
func Login(c *gin.Context) {
	// 1. 解析请求参数
	var req dto.AuthRequest
	if !bindJSON(c, &req, "login") {
		return
	}

	// 2. 调用 Logic RPC 服务进行登录验证
	userID, userName, token, err := rpc.LogicRPCObj.Login(req.Username, req.Password)
	if err != nil {
		clog.Module("user").Warnf("Login failed: username=%s, error=%v", req.Username, err)
		c.JSON(dto.StatusUnauth, dto.ErrorResponse{
			Code:  dto.CodeFail,
			Error: "登录失败，用户名或密码错误",
		})
		return
	}

	clog.Module("user").Infof("Login successful: userID=%d, username=%s", userID, userName)
	// 3. 返回登录成功响应
	c.JSON(dto.StatusOK, dto.LoginResponse{
		Code:     dto.CodeSuccess,
		UserID:   userID,
		UserName: userName,
		Token:    token,
	})
}

// Register 处理用户注册请求
func Register(c *gin.Context) {
	// 1. 解析请求参数
	var req dto.AuthRequest
	if !bindJSON(c, &req, "register") {
		return
	}

	// 2. 调用逻辑服务进行用户注册
	err := rpc.LogicRPCObj.Register(req.Username, req.Password)
	if err != nil {
		clog.Module("user").Warnf("Registration failed: username=%s, error=%v", req.Username, err)
		c.JSON(dto.StatusServerError, dto.ErrorResponse{
			Code:  dto.CodeFail,
			Error: "注册失败，用户名可能已存在",
		})
		return
	}

	clog.Module("user").Infof("Registration successful: username=%s", req.Username)
	c.JSON(dto.StatusOK, dto.RegisterResponse{
		Code: dto.CodeSuccess,
	})
}

// Logout 处理用户登出请求
func Logout(c *gin.Context) {
	var req dto.TokenRequest
	if !bindJSON(c, &req, "logout") {
		return
	}

	// 调用逻辑服务进行登出处理
	err := rpc.LogicRPCObj.Logout(req.Token)
	if err != nil {
		clog.Module("user").Warnf("Logout failed: token=%s, error=%v", req.Token, err)
		c.JSON(dto.StatusServerError, dto.ErrorResponse{
			Code:  dto.CodeFail,
			Error: "登出失败，令牌可能已过期",
		})
		return
	}

	clog.Module("user").Infof("Logout successful: token=%s", req.Token)
	c.JSON(dto.StatusOK, dto.LogoutResponse{Code: dto.CodeSuccess})
}

// CheckAuth 验证用户认证状态
func CheckAuth(c *gin.Context) {
	var req dto.TokenRequest
	if !bindJSON(c, &req, "auth-check") {
		return
	}

	// 调用逻辑服务验证令牌
	err := rpc.LogicRPCObj.CheckAuth(req.Token)
	if err != nil {
		clog.Module("user").Warnf("Authentication failed: token=%s, error=%v", req.Token, err)
		c.JSON(dto.StatusUnauth, dto.ErrorResponse{
			Code:  dto.CodeFail,
			Error: "认证失败，令牌无效或已过期",
		})
		return
	}

	clog.Module("user").Infof("Authentication successful: token=%s", req.Token)
	c.JSON(dto.StatusOK, dto.CheckAuthResponse{
		Code: dto.CodeSuccess,
	})
}
