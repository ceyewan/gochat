package handler

import (
	"time"

	"gochat/api/rpc"
	"gochat/clog"
	"gochat/config"

	"github.com/gin-gonic/gin"
)

// 消息推送请求结构体
type PushForm struct {
	Msg       string `json:"msg"`       // 消息内容
	ToUserId  int    `json:"toUserId"`  // 接收用户ID（单聊时使用）
	RoomId    int    `json:"roomId"`    // 房间ID
	AuthToken string `json:"authToken"` // 认证令牌
}

// 获取当前时间的格式化字符串
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// 处理认证并返回用户信息
func authenticateUser(c *gin.Context, token string) (bool, int, string) {
	code, userID, userName := rpc.LogicRPCObj.CheckAuth(token)
	if code != config.RPCCodeSuccess {
		clog.Warning("Authentication failed: token=%s, code=%d", token, code)
		c.JSON(statusUnauth, response{
			Code:  code,
			Error: "认证失败",
		})
		return false, 0, ""
	}
	return true, userID, userName
}

// Push 处理单聊消息推送
func Push(c *gin.Context) {
	var req PushForm
	if !bindJSON(c, &req, "push") {
		return
	}

	// 认证用户
	ok, userID, userName := authenticateUser(c, req.AuthToken)
	if !ok {
		return
	}

	// 获取目标用户名称
	code, toUserName := rpc.LogicRPCObj.GetUserNameByID(req.ToUserId)
	if code != config.RPCCodeSuccess {
		clog.Warning("Failed to get user info: toUserID=%d, code=%d", req.ToUserId, code)
		c.JSON(statusServerError, response{
			Code:  code,
			Error: "获取用户信息失败",
		})
		return
	}

	// 发送消息
	pushCode := rpc.LogicRPCObj.Push(
		userID, req.ToUserId, req.RoomId,
		config.OpSingleSend, req.Msg,
		userName, toUserName, getCurrentTime(),
	)

	if pushCode != config.RPCCodeSuccess {
		clog.Error("Message sending failed: from=%d, to=%d, code=%d",
			userID, req.ToUserId, pushCode)
		c.JSON(statusServerError, response{
			Code:  pushCode,
			Error: "发送消息失败",
		})
		return
	}

	clog.Info("Message sent: from=%d(%s), to=%d(%s), room=%d",
		userID, userName, req.ToUserId, toUserName, req.RoomId)
	c.JSON(statusOK, response{Code: pushCode})
}

// PushRoom 处理群聊消息推送
func PushRoom(c *gin.Context) {
	var req PushForm
	if !bindJSON(c, &req, "push-room") {
		return
	}

	// 认证用户
	ok, userID, userName := authenticateUser(c, req.AuthToken)
	if !ok {
		return
	}

	// 发送群消息
	pushCode := rpc.LogicRPCObj.PushRoom(
		userID, req.RoomId, config.OpRoomSend,
		req.Msg, userName, getCurrentTime(),
	)

	if pushCode != config.RPCCodeSuccess {
		clog.Error("Room message sending failed: from=%d, room=%d, code=%d",
			userID, req.RoomId, pushCode)
		c.JSON(statusServerError, response{
			Code:  pushCode,
			Error: "发送消息失败",
		})
		return
	}

	clog.Info("Room message sent: from=%d(%s), room=%d",
		userID, userName, req.RoomId)
	c.JSON(statusOK, response{Code: pushCode})
}
