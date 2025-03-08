package handler

import (
	"gochat/api/rpc"
	"gochat/clog"
	"gochat/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthRequest struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

func Login(c *gin.Context) {
	var req AuthRequest
	if err := c.BindJSON(&req); err != nil {
		clog.Info("Login bind json failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 调用逻辑服务的RPC方法
	code, token := rpc.LogicRPCObj.Login(req.UserName, req.Password)
	if code != config.RPCCodeSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"code": code, "error": "登录失败"})
		return
	}
	clog.Info("Login success, name: %s, token: %s", req.UserName, token)
	c.JSON(http.StatusOK, gin.H{"code": code, "token": token})
}

func Register(c *gin.Context) {
	var req AuthRequest
	if err := c.BindJSON(&req); err != nil {
		clog.Info("Register bind json failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 调用逻辑服务的RPC方法
	code, token := rpc.LogicRPCObj.Register(req.UserName, req.Password)
	if code != config.RPCCodeSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"code": code, "error": "注册失败"})
		return
	}
	clog.Info("Register success, name: %s, token: %s", req.UserName, token)
	c.JSON(http.StatusOK, gin.H{"code": code, "token": token})
}

type AuthToken struct {
	Token string `json:"token"`
}

func Logout(c *gin.Context) {
	var req AuthToken
	if err := c.BindJSON(&req); err != nil {
		clog.Info("Logout bind json failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 调用逻辑服务的RPC方法
	code := rpc.LogicRPCObj.Logout(req.Token)
	if code != config.RPCCodeSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"code": code, "error": "登出失败"})
		return
	}
	clog.Info("Logout success, token: %s", req.Token)
	c.JSON(http.StatusOK, gin.H{"code": code})
}

func CheckAuth(c *gin.Context) {
	var req AuthToken
	if err := c.BindJSON(&req); err != nil {
		clog.Info("CheckAuth bind json failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 调用逻辑服务的RPC方法
	code, userID, userName := rpc.LogicRPCObj.CheckAuth(req.Token)
	if code != config.RPCCodeSuccess {
		c.JSON(http.StatusUnauthorized, gin.H{"code": code, "error": "认证失败"})
		return
	}
	// 将用户ID和用户名打包为 JSON 返回
	clog.Info("CheckAuth success, token: %s, userID: %d, userName: %s", req.Token, userID, userName)
	c.JSON(http.StatusOK, gin.H{"code": code, "userid": userID, "username": userName})
}
