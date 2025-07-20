package handler

import (
	"gochat/api/dto"
	"gochat/api/rpc"
	"gochat/clog"

	"github.com/gin-gonic/gin"
)

// Push 处理单聊消息推送
func Push(c *gin.Context) {
	var req dto.PushRequest
	if !bindJSON(c, &req, "push") {
		return
	}

	// 发送消息
	err := rpc.LogicRPCObj.Push(&req)
	if err != nil {
		clog.Module("push").Errorf("Message sending failed: from=%d, to=%d, error=%v",
			req.FromUserID, req.ToUserID, err)
		c.JSON(dto.StatusServerError, dto.ErrorResponse{
			Code:  dto.CodeFail,
			Error: "发送消息失败",
		})
		return
	}

	clog.Module("push").Infof("Message sent: from=%d(%s), to=%d(%s), room=%d",
		req.FromUserID, req.FromUserName, req.ToUserID, req.ToUserName, req.RoomID)
	c.JSON(dto.StatusOK, dto.PushResponse{Code: dto.CodeSuccess})
}

// PushRoom 处理群聊消息推送
func PushRoom(c *gin.Context) {
	var req dto.PushRequest
	if !bindJSON(c, &req, "push-room") {
		return
	}

	// 发送群消息
	err := rpc.LogicRPCObj.PushRoom(&req)

	if err != nil {
		clog.Module("push").Errorf("Room message sending failed: from=%d, room=%d, error=%v",
			req.FromUserID, req.RoomID, err)
		c.JSON(dto.StatusServerError, dto.ErrorResponse{
			Code:  dto.CodeFail,
			Error: "发送消息失败",
		})
		return
	}

	clog.Module("push").Infof("Room message sent: from=%d(%s), room=%d",
		req.FromUserID, req.FromUserName, req.RoomID)
	c.JSON(dto.StatusOK, dto.PushResponse{Code: dto.CodeSuccess})
}
