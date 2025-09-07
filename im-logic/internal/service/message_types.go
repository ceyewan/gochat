package service

import (
	logicpb "github.com/ceyewan/gochat/api/gen/im_logic/v1"
)

// 这些类型在 im_logic/v1/message.proto 中定义，但为了完整性，我们在这里提供实现

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	UserId         string              `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	ConversationId string              `protobuf:"bytes,2,opt,name=conversation_id,json=conversationId,proto3" json:"conversation_id,omitempty"`
	Type           logicpb.MessageType `protobuf:"varint,3,opt,name=type,proto3,enum=im.logic.v1.MessageType" json:"type,omitempty"`
	Content        string              `protobuf:"bytes,4,opt,name=content,proto3" json:"content,omitempty"`
	ClientMsgId    string              `protobuf:"bytes,5,opt,name=client_msg_id,json=clientMsgId,proto3" json:"client_msg_id,omitempty"`
	Extra          string              `protobuf:"bytes,6,opt,name=extra,proto3" json:"extra,omitempty"`
}

// SendMessageResponse 发送消息响应
type SendMessageResponse struct {
	MessageId string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	SeqId     int64  `protobuf:"varint,2,opt,name=seq_id,json=seqId,proto3" json:"seq_id,omitempty"`
	Success   bool   `protobuf:"varint,3,opt,name=success,proto3" json:"success,omitempty"`
}

// GetMessageRequest 获取消息请求
type GetMessageRequest struct {
	MessageId string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
}

// GetMessageResponse 获取消息响应
type GetMessageResponse struct {
	Message *logicpb.Message `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}

// DeleteMessageRequest 删除消息请求
type DeleteMessageRequest struct {
	MessageId  string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	OperatorId string `protobuf:"bytes,2,opt,name=operator_id,json=operatorId,proto3" json:"operator_id,omitempty"`
	Reason     string `protobuf:"bytes,3,opt,name=reason,proto3" json:"reason,omitempty"`
}

// DeleteMessageResponse 删除消息响应
type DeleteMessageResponse struct {
	Success bool `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
}

// UpdateMessageRequest 更新消息请求
type UpdateMessageRequest struct {
	MessageId  string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	OperatorId string `protobuf:"bytes,2,opt,name=operator_id,json=operatorId,proto3" json:"operator_id,omitempty"`
	NewContent string `protobuf:"bytes,3,opt,name=new_content,json=newContent,proto3" json:"new_content,omitempty"`
}

// UpdateMessageResponse 更新消息响应
type UpdateMessageResponse struct {
	Success bool             `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Message *logicpb.Message `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

// ForwardMessageRequest 转发消息请求
type ForwardMessageRequest struct {
	UserId               string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	SourceConversationId string `protobuf:"bytes,2,opt,name=source_conversation_id,json=sourceConversationId,proto3" json:"source_conversation_id,omitempty"`
	TargetConversationId string `protobuf:"bytes,3,opt,name=target_conversation_id,json=targetConversationId,proto3" json:"target_conversation_id,omitempty"`
	MessageId            string `protobuf:"bytes,4,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
}

// ForwardMessageResponse 转发消息响应
type ForwardMessageResponse struct {
	MessageId string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	SeqId     int64  `protobuf:"varint,2,opt,name=seq_id,json=seqId,proto3" json:"seq_id,omitempty"`
	Success   bool   `protobuf:"varint,3,opt,name=success,proto3" json:"success,omitempty"`
}
