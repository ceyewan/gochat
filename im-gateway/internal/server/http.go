package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-gateway/internal/config"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/gin-gonic/gin"
)

// HTTPServer HTTP 服务器
type HTTPServer struct {
	config *config.Config
	server *gin.Engine
	logger clog.Logger
}

// NewHTTPServer 创建 HTTP 服务器
func NewHTTPServer(cfg *config.Config) *HTTPServer {
	// 设置 Gin 模式
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	server := gin.New()

	// 添加中间件
	server.Use(gin.Recovery())
	server.Use(loggingMiddleware(cfg.Log))
	server.Use(corsMiddleware())

	return &HTTPServer{
		config: cfg,
		server: server,
		logger: clog.Module("http-server"),
	}
}

// Engine 返回 gin 引擎实例
func (h *HTTPServer) Engine() *gin.Engine {
	return h.server
}

// RegisterRoutes 注册路由
func (h *HTTPServer) RegisterRoutes() {
	// 健康检查
	h.server.GET("/health", h.healthCheck)

	// API v1 路由组
	v1 := h.server.Group("/api/v1")
	{
		// 认证相关路由
		auth := v1.Group("/auth")
		{
			auth.POST("/login", h.handleLogin)
			auth.POST("/register", h.handleRegister)
			auth.POST("/refresh", h.handleRefreshToken)
			auth.POST("/logout", h.handleLogout)
		}

		// 用户相关路由
		users := v1.Group("/users")
		{
			users.GET("/info", h.authMiddleware(), h.getUserInfo)
			users.GET("/:username", h.authMiddleware(), h.searchUser)
		}

		// 会话相关路由
		conversations := v1.Group("/conversations")
		{
			conversations.GET("", h.authMiddleware(), h.getConversations)
			conversations.GET("/:id/messages", h.authMiddleware(), h.getConversationMessages)
			conversations.PUT("/:id/read", h.authMiddleware(), h.markConversationRead)
			conversations.POST("", h.authMiddleware(), h.createConversation)
		}

		// 好友相关路由
		friends := v1.Group("/friends")
		{
			friends.POST("", h.authMiddleware(), h.addFriend)
			friends.GET("", h.authMiddleware(), h.getFriends)
		}

		// 群组相关路由
		groups := v1.Group("/groups")
		{
			groups.POST("", h.authMiddleware(), h.createGroup)
			groups.GET("/:id", h.authMiddleware(), h.getGroup)
		}

		// 消息相关路由
		messages := v1.Group("/messages")
		{
			messages.POST("", h.authMiddleware(), h.sendMessage)
		}

		// 文件上传相关路由
		uploads := v1.Group("/uploads")
		{
			uploads.POST("", h.authMiddleware(), h.requestUploadURL)
		}

		// 搜索相关路由
		search := v1.Group("/search")
		{
			search.GET("", h.authMiddleware(), h.search)
		}
	}
}

// healthCheck 健康检查
func (h *HTTPServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

// authMiddleware JWT 认证中间件
func (h *HTTPServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "缺少认证头",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// 提取 Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "无效的认证格式",
				"code":    "INVALID_TOKEN_FORMAT",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// TODO: 调用 im-logic 的 ValidateToken 接口验证 token
		// 这里暂时跳过验证，实际实现时需要调用 gRPC 接口

		// 验证通过，继续处理请求
		c.Next()
	}
}

// loggingMiddleware 日志中间件
func loggingMiddleware(logConfig config.LogConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		// 记录请求日志
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		logger := clog.Module("http-request")
		logger.Info("HTTP请求",
			clog.String("method", method),
			clog.String("path", path),
			clog.Int("status", statusCode),
			clog.Duration("latency", latency),
			clog.String("client_ip", c.ClientIP()),
		)
	}
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// 统一响应格式
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Code    string      `json:"code,omitempty"`
}

// successResponse 成功响应
func (h *HTTPServer) successResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// errorResponse 错误响应
func (h *HTTPServer) errorResponse(c *gin.Context, statusCode int, message string, code string) {
	c.JSON(statusCode, Response{
		Success: false,
		Message: message,
		Code:    code,
	})
}

// 以下是各个处理函数的框架实现

func (h *HTTPServer) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "请求参数错误", "INVALID_PARAMS")
		return
	}

	// TODO: 调用 im-logic 的 Login 接口
	h.successResponse(c, "登录成功", gin.H{
		"access_token":  "mock_token",
		"refresh_token": "mock_refresh_token",
		"expires_in":    3600,
		"user": gin.H{
			"id":       "user123",
			"username": req.Username,
			"nickname": "测试用户",
		},
	})
}

func (h *HTTPServer) handleRegister(c *gin.Context) {
	var req struct {
		Username  string `json:"username" binding:"required"`
		Password  string `json:"password" binding:"required"`
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "请求参数错误", "INVALID_PARAMS")
		return
	}

	// TODO: 调用 im-logic 的 Register 接口
	h.successResponse(c, "注册成功", gin.H{
		"id":       "user123",
		"username": req.Username,
		"nickname": req.Nickname,
	})
}

func (h *HTTPServer) handleRefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "请求参数错误", "INVALID_PARAMS")
		return
	}

	// TODO: 调用 im-logic 的 RefreshToken 接口
	h.successResponse(c, "令牌刷新成功", gin.H{
		"access_token":  "new_mock_token",
		"refresh_token": "new_mock_refresh_token",
		"expires_in":    3600,
	})
}

func (h *HTTPServer) handleLogout(c *gin.Context) {
	// TODO: 调用 im-logic 的 Logout 接口
	h.successResponse(c, "登出成功", nil)
}

func (h *HTTPServer) getUserInfo(c *gin.Context) {
	// TODO: 调用 im-logic 获取用户信息
	h.successResponse(c, "获取用户信息成功", gin.H{
		"id":         "user123",
		"username":   "testuser",
		"nickname":   "测试用户",
		"avatar_url": "",
		"is_guest":   false,
		"created_at": time.Now().Unix(),
	})
}

func (h *HTTPServer) searchUser(c *gin.Context) {
	username := c.Param("username")
	// TODO: 调用 im-logic 搜索用户
	h.successResponse(c, "搜索用户成功", gin.H{
		"id":         "user123",
		"username":   username,
		"nickname":   "测试用户",
		"avatar_url": "",
	})
}

func (h *HTTPServer) getConversations(c *gin.Context) {
	// TODO: 调用 im-logic 获取会话列表
	h.successResponse(c, "获取会话列表成功", []gin.H{})
}

func (h *HTTPServer) getConversationMessages(c *gin.Context) {
	conversationID := c.Param("id")
	// TODO: 调用 im-logic 获取会话消息
	h.successResponse(c, "获取会话消息成功", gin.H{
		"conversation_id": conversationID,
		"messages":        []gin.H{},
	})
}

func (h *HTTPServer) markConversationRead(c *gin.Context) {
	conversationID := c.Param("id")
	// TODO: 调用 im-logic 标记会话已读
	h.successResponse(c, "标记已读成功", nil)
}

func (h *HTTPServer) createConversation(c *gin.Context) {
	var req struct {
		TargetUserID string `json:"target_user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "请求参数错误", "INVALID_PARAMS")
		return
	}

	// TODO: 调用 im-logic 创建会话
	h.successResponse(c, "创建会话成功", gin.H{
		"conversation_id": "conv123",
	})
}

func (h *HTTPServer) addFriend(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "请求参数错误", "INVALID_PARAMS")
		return
	}

	// TODO: 调用 im-logic 添加好友
	h.successResponse(c, "添加好友成功", nil)
}

func (h *HTTPServer) getFriends(c *gin.Context) {
	// TODO: 调用 im-logic 获取好友列表
	h.successResponse(c, "获取好友列表成功", []gin.H{})
}

func (h *HTTPServer) createGroup(c *gin.Context) {
	var req struct {
		Name    string   `json:"name" binding:"required"`
		UserIDs []string `json:"user_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "请求参数错误", "INVALID_PARAMS")
		return
	}

	// TODO: 调用 im-logic 创建群组
	h.successResponse(c, "创建群组成功", gin.H{
		"group_id": "group123",
	})
}

func (h *HTTPServer) getGroup(c *gin.Context) {
	groupID := c.Param("id")
	// TODO: 调用 im-logic 获取群组信息
	h.successResponse(c, "获取群组信息成功", gin.H{
		"id":   groupID,
		"name": "测试群组",
	})
}

func (h *HTTPServer) sendMessage(c *gin.Context) {
	var req struct {
		ConversationID string `json:"conversation_id" binding:"required"`
		MessageType    int    `json:"message_type" binding:"required"`
		Content        string `json:"content" binding:"required"`
		ClientMsgID    string `json:"client_msg_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "请求参数错误", "INVALID_PARAMS")
		return
	}

	// TODO: 调用 im-logic 发送消息
	h.successResponse(c, "消息发送成功", gin.H{
		"message_id": "msg123",
	})
}

func (h *HTTPServer) requestUploadURL(c *gin.Context) {
	// TODO: 实现文件上传凭证请求
	h.successResponse(c, "获取上传凭证成功", gin.H{
		"upload_url":   "https://example.com/upload",
		"download_url": "https://example.com/download",
		"fields":       gin.H{},
	})
}

func (h *HTTPServer) search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		h.errorResponse(c, http.StatusBadRequest, "缺少搜索关键词", "MISSING_QUERY")
		return
	}

	// TODO: 调用 im-logic 搜索接口
	h.successResponse(c, "搜索成功", gin.H{
		"results": []gin.H{},
		"total":   0,
	})
}
