package handler

import (
	"gochat/api/rpc"
	"gochat/clog"
	"gochat/config"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type PushForm struct {
	Msg       string `json:"msg"`
	ToUserId  int    `json:"toUserId"`
	RoomId    int    `json:"roomId"`
	AuthToken string `json:"authToken"`
}

func Push(c *gin.Context) {
	var req PushForm
	if err := c.BindJSON(&req); err != nil {
		clog.Error("Push bind json failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户身份
	code, userID, userName := rpc.LogicRPCObj.CheckAuth(req.AuthToken)
	if code != config.RPCCodeSuccess {
		c.JSON(http.StatusUnauthorized, gin.H{"code": code, "error": "认证失败"})
		return
	}

	// 获取目标用户名称
	code, toUserName := rpc.LogicRPCObj.GetUserNameByID(req.ToUserId)
	if code != config.RPCCodeSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"code": code, "error": "获取用户信息失败"})
		return
	}

	// 获取当前时间作为消息创建时间
	createTime := time.Now().Format("2006-01-02 15:04:05")

	// 调用逻辑服务的RPC方法发送消息
	pushCode := rpc.LogicRPCObj.Push(
		userID,              // 发送者ID
		req.ToUserId,        // 接收者ID
		req.RoomId,          // 房间ID
		config.OpSingleSend, // 操作类型：单聊
		req.Msg,             // 消息内容
		userName,            // 发送者用户名
		toUserName,          // 接收者用户名
		createTime,          // 创建时间
	)

	if pushCode != config.RPCCodeSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"code": pushCode, "error": "发送消息失败"})
		return
	}

	clog.Info("Push success, from: %d(%s), to: %d(%s), msg: %s",
		userID, userName, req.ToUserId, toUserName, req.Msg)
	c.JSON(http.StatusOK, gin.H{"code": pushCode})
}

func PushRoom(c *gin.Context) {
	var req PushForm
	if err := c.BindJSON(&req); err != nil {
		clog.Error("PushRoom bind json failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户身份
	code, userID, userName := rpc.LogicRPCObj.CheckAuth(req.AuthToken)
	if code != config.RPCCodeSuccess {
		c.JSON(http.StatusUnauthorized, gin.H{"code": code, "error": "认证失败"})
		return
	}

	// 获取当前时间作为消息创建时间
	createTime := time.Now().Format("2006-01-02 15:04:05")

	// 调用逻辑服务的RPC方法发送消息
	pushCode := rpc.LogicRPCObj.PushRoom(
		userID,            // 发送者ID
		req.RoomId,        // 房间ID
		config.OpRoomSend, // 操作类型：群聊
		req.Msg,           // 消息内容
		userName,          // 发送者用户名
		createTime,        // 创建时间
	)

	if pushCode != config.RPCCodeSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"code": pushCode, "error": "发送消息失败"})
		return
	}

	clog.Info("PushRoom success, from: %d(%s), room: %d, msg: %s",
		userID, userName, req.RoomId, req.Msg)
	c.JSON(http.StatusOK, gin.H{"code": pushCode})
}

type FormCount struct {
	RoomId int `json:"roomId" binding:"required"`
}

// Count 统计房间消息数量，调用 Send
func Count(c *gin.Context) {
	var req FormCount
	if err := c.Bind(&req); err != nil {
		clog.Error("Count bind json failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用逻辑服务的RPC方法统计消息数量
	countCode := rpc.LogicRPCObj.Count(req.RoomId, config.OpRoomCountSend)
	if countCode != config.RPCCodeSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"code": countCode, "error": "统计消息数量失败"})
		return
	}

	clog.Info("Count success, room: %d", req.RoomId)
	c.JSON(http.StatusOK, gin.H{"code": countCode})
}

func GetRoomInfo(c *gin.Context) {
	var req FormCount
	if err := c.Bind(&req); err != nil {
		clog.Error("GetRoomInfo bind json failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用逻辑服务的RPC方法获取房间信息
	code := rpc.LogicRPCObj.GetRoomInfo(req.RoomId, config.OpRoomInfoSend)
	if code != config.RPCCodeSuccess {
		c.JSON(http.StatusInternalServerError, gin.H{"code": code, "error": "获取房间信息失败"})
		return
	}

	clog.Info("GetRoomInfo success, room: %d", req.RoomId)
	c.JSON(http.StatusOK, gin.H{"code": code})
}
