package handler

import (
	"net/http"

	"gochat/api/rpc"
	"gochat/clog"
	"gochat/config"

	"github.com/gin-gonic/gin"
)

// 定义通用响应状态码
const (
	statusOK          = http.StatusOK
	statusBadRequest  = http.StatusBadRequest
	statusUnauth      = http.StatusUnauthorized
	statusServerError = http.StatusInternalServerError
)

// 用户认证相关请求结构
type (
	// AuthRequest 用户认证请求结构
	AuthRequest struct {
		UserName string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	// TokenRequest 令牌请求结构
	TokenRequest struct {
		Token string `json:"authToken" binding:"required"`
	}

	// 统一响应结构
	response struct {
		Code  int         `json:"code"`
		Error string      `json:"error,omitempty"`
		Data  interface{} `json:"data,omitempty"`
	}
)

// 处理请求参数绑定的通用函数
func bindJSON(c *gin.Context, req interface{}, action string) bool {
	if err := c.BindJSON(req); err != nil {
		clog.Warning("Failed to parse %s request: %v", action, err)
		c.JSON(statusBadRequest, response{
			Error: "无效的请求参数",
		})
		return false
	}
	return true
}

// Login 处理用户登录请求
func Login(c *gin.Context) {
	var req AuthRequest
	if !bindJSON(c, &req, "login") {
		return
	}

	// 调用逻辑服务进行登录验证
	code, token := rpc.LogicRPCObj.Login(req.UserName, req.Password)
	if code != config.RPCCodeSuccess {
		clog.Warning("Login failed: username=%s, code=%d", req.UserName, code)
		c.JSON(statusUnauth, response{
			Code:  code,
			Error: "登录失败，用户名或密码错误",
		})
		return
	}

	clog.Info("Login successful: username=%s", req.UserName)
	c.JSON(statusOK, response{
		Code: code,
		Data: gin.H{"token": token},
	})
}

// Register 处理用户注册请求
func Register(c *gin.Context) {
	var req AuthRequest
	if !bindJSON(c, &req, "register") {
		return
	}

	// 调用逻辑服务进行用户注册
	code := rpc.LogicRPCObj.Register(req.UserName, req.Password)
	if code != config.RPCCodeSuccess {
		clog.Warning("Registration failed: username=%s, code=%d", req.UserName, code)
		c.JSON(statusServerError, response{
			Code:  code,
			Error: "注册失败，用户名可能已存在",
		})
		return
	}

	clog.Info("Registration successful: username=%s", req.UserName)
	c.JSON(statusOK, response{
		Code: code,
		Data: gin.H{"username": req.UserName},
	})
}

// Logout 处理用户登出请求
func Logout(c *gin.Context) {
	var req TokenRequest
	if !bindJSON(c, &req, "logout") {
		return
	}

	// 调用逻辑服务进行登出处理
	code := rpc.LogicRPCObj.Logout(req.Token)
	if code != config.RPCCodeSuccess {
		clog.Warning("Logout failed: token=%s, code=%d", req.Token, code)
		c.JSON(statusServerError, response{
			Code:  code,
			Error: "登出失败，令牌可能已过期",
		})
		return
	}

	clog.Info("Logout successful: token=%s", req.Token)
	c.JSON(statusOK, response{Code: code})
}

// CheckAuth 验证用户认证状态
func CheckAuth(c *gin.Context) {
	var req TokenRequest
	if !bindJSON(c, &req, "auth-check") {
		return
	}

	// 调用逻辑服务验证令牌
	code, userID, userName := rpc.LogicRPCObj.CheckAuth(req.Token)
	if code != config.RPCCodeSuccess {
		clog.Warning("Authentication failed: token=%s, code=%d", req.Token, code)
		c.JSON(statusUnauth, response{
			Code:  code,
			Error: "认证失败，令牌无效或已过期",
		})
		return
	}

	clog.Info("Authentication successful: userID=%d, username=%s", userID, userName)
	c.JSON(statusOK, response{
		Code: code,
		Data: gin.H{
			"userid":   userID,
			"username": userName,
		},
	})
}
