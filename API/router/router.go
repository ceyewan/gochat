package router

import (
	"gochat/api/handler"
	"gochat/api/rpc"
	"gochat/clog"
	"gochat/config"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Register 初始化并返回配置好的 Gin 引擎实例
//
// 该函数完成以下工作:
//  1. 创建一个默认的 Gin 引擎实例
//  2. 注册跨域中间件
//  3. 初始化用户相关路由
//  4. 初始化消息推送相关路由
//  5. 配置 404 处理
//
// 返回:
//   - *gin.Engine: 配置好的 Gin 引擎实例
func Register() *gin.Engine {
	r := gin.Default()
	r.Use(CorsMiddleware())
	initUserRouter(r)
	initPushRouter(r)
	r.NoRoute(func(c *gin.Context) {
		clog.Info("404 Not Found")
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "error": "404 Not Found"})
	})
	return r
}

// initUserRouter 初始化用户相关的路由
//
// 注册与用户操作相关的 API 端点，包括:
//   - 登录
//   - 注册
//   - 身份验证
//   - 登出
//
// 除登录和注册外，其他端点需要会话验证
//
// 参数:
//   - r: Gin 引擎实例
func initUserRouter(r *gin.Engine) {
	userGroup := r.Group("/user")
	userGroup.POST("/login", handler.Login)
	userGroup.POST("/register", handler.Register)
	userGroup.Use(CheckSessionId())
	{
		userGroup.POST("/checkAuth", handler.CheckAuth)
		userGroup.POST("/logout", handler.Logout)
	}

}

// initPushRouter 初始化消息推送相关的路由
//
// 注册与消息推送相关的 API 端点，包括:
//   - 单聊消息推送
//   - 群聊消息推送
//   - 获取房间人数
//   - 获取房间信息
//
// 所有端点都需要会话验证
//
// 参数:
//   - r: Gin 引擎实例
func initPushRouter(r *gin.Engine) {
	pushGroup := r.Group("/push")
	pushGroup.Use(CheckSessionId())
	{
		pushGroup.POST("/push", handler.Push)
		pushGroup.POST("/pushRoom", handler.PushRoom)
		pushGroup.POST("/count", handler.Count)
		pushGroup.POST("/getRoomInfo", handler.GetRoomInfo)
	}

}

// FormCheckSessionId 定义了会话验证的请求参数结构
//
// 包含以下字段:
//   - AuthToken: 用户身份验证令牌
type FormCheckSessionId struct {
	AuthToken string `form:"authToken" json:"authToken" binding:"required"`
}

// CheckSessionId 返回一个用于验证用户会话有效性的中间件
//
// 该中间件从请求体中提取 AuthToken，并通过 RPC 调用验证其有效性
// 如果验证失败，则中止请求处理并返回会话错误响应
// 如果验证成功，则继续处理后续中间件或处理函数
//
// 返回:
//   - gin.HandlerFunc: Gin 中间件函数
func CheckSessionId() gin.HandlerFunc {
	return func(c *gin.Context) {
		var formCheckSessionId FormCheckSessionId
		if err := c.ShouldBindBodyWith(&formCheckSessionId, binding.JSON); err != nil {
			clog.Error("CheckSessionId bind json failed: %v", err)
			c.Abort()
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		authToken := formCheckSessionId.AuthToken
		code, userId, userName := rpc.LogicRPCObj.CheckAuth(authToken)

		if code != config.RPCCodeSuccess || userId <= 0 || userName == "" {
			clog.Error("验证会话失败, authToken: %s, code: %d, userId: %d, userName: %s",
				authToken, code, userId, userName)
			c.Abort()
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":  config.RPCSessionError,
				"error": "会话无效",
			})
			return
		}

		// 验证成功，将用户信息添加到上下文中以便后续处理函数使用
		c.Set("userId", userId)
		c.Set("userName", userName)
		c.Set("authToken", authToken)

		c.Next()
	}
}

// CorsMiddleware 返回一个处理跨域资源共享(CORS)的中间件
//
// 该中间件为响应添加必要的 CORS 头，允许跨域请求
// 对于 OPTIONS 请求方法，直接返回成功响应
//
// 返回:
//   - gin.HandlerFunc: Gin 中间件函数
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		var openCorsFlag = true
		if openCorsFlag {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS, POST, PUT, DELETE")
			c.Set("content-type", "application/json")
		}
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, nil)
		}
		c.Next()
	}
}
