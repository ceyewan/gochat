package router

import (
	"gochat/api/dto"
	"net/http"

	"gochat/api/handler"
	"gochat/api/rpc"
	"gochat/clog"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Register 初始化并返回配置好的Gin引擎实例
func Register() *gin.Engine {
	r := gin.Default()
	r.Use(CorsMiddleware())

	// 初始化各模块路由
	initUserRouter(r)
	initPushRouter(r)

	// 处理404请求
	r.NoRoute(func(c *gin.Context) {
		clog.Module("router").Infof("404 Not Found: %s %s", c.Request.Method, c.Request.URL.Path)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "error": "404 Not Found"})
	})

	return r
}

// initUserRouter 初始化用户相关路由
func initUserRouter(r *gin.Engine) {
	userGroup := r.Group("/user")

	// 无需验证的路由
	userGroup.POST("/login", handler.Login)
	userGroup.POST("/register", handler.Register)
	userGroup.POST("/checkAuth", handler.CheckAuth)
	userGroup.POST("/logout", handler.Logout)
}

// initPushRouter 初始化消息推送相关路由
func initPushRouter(r *gin.Engine) {
	pushGroup := r.Group("/push")
	// 需要会话验证的路由
	pushGroup.Use(CheckSessionId())
	{
		pushGroup.POST("/push", handler.Push)
		pushGroup.POST("/pushRoom", handler.PushRoom)
	}
}

// CheckSessionId 返回验证用户会话有效性的中间件
func CheckSessionId() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.TokenRequest

		// 解析请求体中的会话信息
		if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
			clog.Module("router").Warnf("Session check failed: invalid request format: %v", err)
			c.Abort()
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话请求格式"})
			return
		}

		// 验证会话是否有效
		authToken := req.Token
		err := rpc.LogicRPCObj.CheckAuth(authToken)

		if err != nil {
			clog.Module("router").Warnf("Session check failed: token=%s, error=%v", authToken, err)
			c.Abort()
			c.JSON(http.StatusUnauthorized, gin.H{"error": "会话无效"})
			return
		}
		clog.Module("router").Debugf("Session check success: token=%s", authToken)
		c.Next()
	}
}

// CorsMiddleware 返回处理跨域资源共享(CORS)的中间件
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		// 添加CORS相关响应头
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS, POST, PUT, DELETE")
		c.Set("content-type", "application/json")

		// 对OPTIONS请求直接返回成功
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
