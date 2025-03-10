// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        v4.25.3
// source: logic.proto

package logicproto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// 用户认证相关消息
type RegisterRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Password      string                 `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RegisterRequest) Reset() {
	*x = RegisterRequest{}
	mi := &file_logic_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RegisterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RegisterRequest) ProtoMessage() {}

func (x *RegisterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RegisterRequest.ProtoReflect.Descriptor instead.
func (*RegisterRequest) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{0}
}

func (x *RegisterRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *RegisterRequest) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type RegisterReply struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Code          int32                  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RegisterReply) Reset() {
	*x = RegisterReply{}
	mi := &file_logic_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RegisterReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RegisterReply) ProtoMessage() {}

func (x *RegisterReply) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RegisterReply.ProtoReflect.Descriptor instead.
func (*RegisterReply) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{1}
}

func (x *RegisterReply) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

type LoginRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Password      string                 `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoginRequest) Reset() {
	*x = LoginRequest{}
	mi := &file_logic_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoginRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginRequest) ProtoMessage() {}

func (x *LoginRequest) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginRequest.ProtoReflect.Descriptor instead.
func (*LoginRequest) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{2}
}

func (x *LoginRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *LoginRequest) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type LoginResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Code          int32                  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	AuthToken     string                 `protobuf:"bytes,2,opt,name=auth_token,json=authToken,proto3" json:"auth_token,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoginResponse) Reset() {
	*x = LoginResponse{}
	mi := &file_logic_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoginResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginResponse) ProtoMessage() {}

func (x *LoginResponse) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginResponse.ProtoReflect.Descriptor instead.
func (*LoginResponse) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{3}
}

func (x *LoginResponse) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *LoginResponse) GetAuthToken() string {
	if x != nil {
		return x.AuthToken
	}
	return ""
}

type CheckAuthRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	AuthToken     string                 `protobuf:"bytes,1,opt,name=auth_token,json=authToken,proto3" json:"auth_token,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CheckAuthRequest) Reset() {
	*x = CheckAuthRequest{}
	mi := &file_logic_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CheckAuthRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CheckAuthRequest) ProtoMessage() {}

func (x *CheckAuthRequest) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CheckAuthRequest.ProtoReflect.Descriptor instead.
func (*CheckAuthRequest) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{4}
}

func (x *CheckAuthRequest) GetAuthToken() string {
	if x != nil {
		return x.AuthToken
	}
	return ""
}

type CheckAuthResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Code          int32                  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	UserId        int32                  `protobuf:"varint,2,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	UserName      string                 `protobuf:"bytes,3,opt,name=user_name,json=userName,proto3" json:"user_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CheckAuthResponse) Reset() {
	*x = CheckAuthResponse{}
	mi := &file_logic_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CheckAuthResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CheckAuthResponse) ProtoMessage() {}

func (x *CheckAuthResponse) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CheckAuthResponse.ProtoReflect.Descriptor instead.
func (*CheckAuthResponse) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{5}
}

func (x *CheckAuthResponse) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *CheckAuthResponse) GetUserId() int32 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *CheckAuthResponse) GetUserName() string {
	if x != nil {
		return x.UserName
	}
	return ""
}

type LogoutRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	AuthToken     string                 `protobuf:"bytes,1,opt,name=auth_token,json=authToken,proto3" json:"auth_token,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LogoutRequest) Reset() {
	*x = LogoutRequest{}
	mi := &file_logic_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LogoutRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogoutRequest) ProtoMessage() {}

func (x *LogoutRequest) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogoutRequest.ProtoReflect.Descriptor instead.
func (*LogoutRequest) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{6}
}

func (x *LogoutRequest) GetAuthToken() string {
	if x != nil {
		return x.AuthToken
	}
	return ""
}

type LogoutResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Code          int32                  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LogoutResponse) Reset() {
	*x = LogoutResponse{}
	mi := &file_logic_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LogoutResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogoutResponse) ProtoMessage() {}

func (x *LogoutResponse) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogoutResponse.ProtoReflect.Descriptor instead.
func (*LogoutResponse) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{7}
}

func (x *LogoutResponse) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

type GetUserInfoRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        int32                  `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetUserInfoRequest) Reset() {
	*x = GetUserInfoRequest{}
	mi := &file_logic_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetUserInfoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetUserInfoRequest) ProtoMessage() {}

func (x *GetUserInfoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetUserInfoRequest.ProtoReflect.Descriptor instead.
func (*GetUserInfoRequest) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{8}
}

func (x *GetUserInfoRequest) GetUserId() int32 {
	if x != nil {
		return x.UserId
	}
	return 0
}

type GetUserInfoResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Code          int32                  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	UserId        int32                  `protobuf:"varint,2,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	UserName      string                 `protobuf:"bytes,3,opt,name=user_name,json=userName,proto3" json:"user_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetUserInfoResponse) Reset() {
	*x = GetUserInfoResponse{}
	mi := &file_logic_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetUserInfoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetUserInfoResponse) ProtoMessage() {}

func (x *GetUserInfoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetUserInfoResponse.ProtoReflect.Descriptor instead.
func (*GetUserInfoResponse) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{9}
}

func (x *GetUserInfoResponse) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *GetUserInfoResponse) GetUserId() int32 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *GetUserInfoResponse) GetUserName() string {
	if x != nil {
		return x.UserName
	}
	return ""
}

// 连接管理相关消息
type ConnectRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	AuthToken     string                 `protobuf:"bytes,1,opt,name=auth_token,json=authToken,proto3" json:"auth_token,omitempty"`
	RoomId        int32                  `protobuf:"varint,2,opt,name=room_id,json=roomId,proto3" json:"room_id,omitempty"`
	ServerId      string                 `protobuf:"bytes,3,opt,name=server_id,json=serverId,proto3" json:"server_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ConnectRequest) Reset() {
	*x = ConnectRequest{}
	mi := &file_logic_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ConnectRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConnectRequest) ProtoMessage() {}

func (x *ConnectRequest) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConnectRequest.ProtoReflect.Descriptor instead.
func (*ConnectRequest) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{10}
}

func (x *ConnectRequest) GetAuthToken() string {
	if x != nil {
		return x.AuthToken
	}
	return ""
}

func (x *ConnectRequest) GetRoomId() int32 {
	if x != nil {
		return x.RoomId
	}
	return 0
}

func (x *ConnectRequest) GetServerId() string {
	if x != nil {
		return x.ServerId
	}
	return ""
}

type ConnectReply struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        int32                  `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ConnectReply) Reset() {
	*x = ConnectReply{}
	mi := &file_logic_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ConnectReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConnectReply) ProtoMessage() {}

func (x *ConnectReply) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConnectReply.ProtoReflect.Descriptor instead.
func (*ConnectReply) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{11}
}

func (x *ConnectReply) GetUserId() int32 {
	if x != nil {
		return x.UserId
	}
	return 0
}

type DisConnectRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        int32                  `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	RoomId        int32                  `protobuf:"varint,2,opt,name=room_id,json=roomId,proto3" json:"room_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DisConnectRequest) Reset() {
	*x = DisConnectRequest{}
	mi := &file_logic_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DisConnectRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DisConnectRequest) ProtoMessage() {}

func (x *DisConnectRequest) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DisConnectRequest.ProtoReflect.Descriptor instead.
func (*DisConnectRequest) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{12}
}

func (x *DisConnectRequest) GetUserId() int32 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *DisConnectRequest) GetRoomId() int32 {
	if x != nil {
		return x.RoomId
	}
	return 0
}

type DisConnectReply struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DisConnectReply) Reset() {
	*x = DisConnectReply{}
	mi := &file_logic_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DisConnectReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DisConnectReply) ProtoMessage() {}

func (x *DisConnectReply) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DisConnectReply.ProtoReflect.Descriptor instead.
func (*DisConnectReply) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{13}
}

// 消息发送相关
type Send struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FromUserId    int32                  `protobuf:"varint,1,opt,name=from_user_id,json=fromUserId,proto3" json:"from_user_id,omitempty"`
	FromUserName  string                 `protobuf:"bytes,2,opt,name=from_user_name,json=fromUserName,proto3" json:"from_user_name,omitempty"`
	ToUserId      int32                  `protobuf:"varint,3,opt,name=to_user_id,json=toUserId,proto3" json:"to_user_id,omitempty"`
	ToUserName    string                 `protobuf:"bytes,4,opt,name=to_user_name,json=toUserName,proto3" json:"to_user_name,omitempty"`
	RoomId        int32                  `protobuf:"varint,5,opt,name=room_id,json=roomId,proto3" json:"room_id,omitempty"`
	Msg           string                 `protobuf:"bytes,6,opt,name=msg,proto3" json:"msg,omitempty"`
	Op            int32                  `protobuf:"varint,7,opt,name=op,proto3" json:"op,omitempty"`
	CreateTime    string                 `protobuf:"bytes,8,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Send) Reset() {
	*x = Send{}
	mi := &file_logic_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Send) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Send) ProtoMessage() {}

func (x *Send) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Send.ProtoReflect.Descriptor instead.
func (*Send) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{14}
}

func (x *Send) GetFromUserId() int32 {
	if x != nil {
		return x.FromUserId
	}
	return 0
}

func (x *Send) GetFromUserName() string {
	if x != nil {
		return x.FromUserName
	}
	return ""
}

func (x *Send) GetToUserId() int32 {
	if x != nil {
		return x.ToUserId
	}
	return 0
}

func (x *Send) GetToUserName() string {
	if x != nil {
		return x.ToUserName
	}
	return ""
}

func (x *Send) GetRoomId() int32 {
	if x != nil {
		return x.RoomId
	}
	return 0
}

func (x *Send) GetMsg() string {
	if x != nil {
		return x.Msg
	}
	return ""
}

func (x *Send) GetOp() int32 {
	if x != nil {
		return x.Op
	}
	return 0
}

func (x *Send) GetCreateTime() string {
	if x != nil {
		return x.CreateTime
	}
	return ""
}

// 通用响应
type SuccessReply struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Code          int32                  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SuccessReply) Reset() {
	*x = SuccessReply{}
	mi := &file_logic_proto_msgTypes[15]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SuccessReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SuccessReply) ProtoMessage() {}

func (x *SuccessReply) ProtoReflect() protoreflect.Message {
	mi := &file_logic_proto_msgTypes[15]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SuccessReply.ProtoReflect.Descriptor instead.
func (*SuccessReply) Descriptor() ([]byte, []int) {
	return file_logic_proto_rawDescGZIP(), []int{15}
}

func (x *SuccessReply) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

var File_logic_proto protoreflect.FileDescriptor

var file_logic_proto_rawDesc = string([]byte{
	0x0a, 0x0b, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x6c,
	0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x41, 0x0a, 0x0f, 0x52, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x22, 0x23, 0x0a, 0x0d,
	0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x12, 0x0a,
	0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x63, 0x6f, 0x64,
	0x65, 0x22, 0x3e, 0x0a, 0x0c, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72,
	0x64, 0x22, 0x42, 0x0a, 0x0d, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x75, 0x74, 0x68, 0x5f, 0x74,
	0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61, 0x75, 0x74, 0x68,
	0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x31, 0x0a, 0x10, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x41, 0x75,
	0x74, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x75, 0x74,
	0x68, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61,
	0x75, 0x74, 0x68, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x5d, 0x0a, 0x11, 0x43, 0x68, 0x65, 0x63,
	0x6b, 0x41, 0x75, 0x74, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x63, 0x6f, 0x64,
	0x65, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x75, 0x73,
	0x65, 0x72, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75,
	0x73, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x2e, 0x0a, 0x0d, 0x4c, 0x6f, 0x67, 0x6f, 0x75,
	0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x75, 0x74, 0x68,
	0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61, 0x75,
	0x74, 0x68, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x24, 0x0a, 0x0e, 0x4c, 0x6f, 0x67, 0x6f, 0x75,
	0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x22, 0x2d, 0x0a,
	0x12, 0x47, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x22, 0x5f, 0x0a, 0x13,
	0x47, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f,
	0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64,
	0x12, 0x1b, 0x0a, 0x09, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x65, 0x0a,
	0x0e, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x1d, 0x0a, 0x0a, 0x61, 0x75, 0x74, 0x68, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x61, 0x75, 0x74, 0x68, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x17,
	0x0a, 0x07, 0x72, 0x6f, 0x6f, 0x6d, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x06, 0x72, 0x6f, 0x6f, 0x6d, 0x49, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x49, 0x64, 0x22, 0x27, 0x0a, 0x0c, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x52,
	0x65, 0x70, 0x6c, 0x79, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x22, 0x45, 0x0a,
	0x11, 0x44, 0x69, 0x73, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x72,
	0x6f, 0x6f, 0x6d, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x72, 0x6f,
	0x6f, 0x6d, 0x49, 0x64, 0x22, 0x11, 0x0a, 0x0f, 0x44, 0x69, 0x73, 0x43, 0x6f, 0x6e, 0x6e, 0x65,
	0x63, 0x74, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0xea, 0x01, 0x0a, 0x04, 0x53, 0x65, 0x6e, 0x64,
	0x12, 0x20, 0x0a, 0x0c, 0x66, 0x72, 0x6f, 0x6d, 0x5f, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x66, 0x72, 0x6f, 0x6d, 0x55, 0x73, 0x65, 0x72,
	0x49, 0x64, 0x12, 0x24, 0x0a, 0x0e, 0x66, 0x72, 0x6f, 0x6d, 0x5f, 0x75, 0x73, 0x65, 0x72, 0x5f,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x66, 0x72, 0x6f, 0x6d,
	0x55, 0x73, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1c, 0x0a, 0x0a, 0x74, 0x6f, 0x5f, 0x75,
	0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x74, 0x6f,
	0x55, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x20, 0x0a, 0x0c, 0x74, 0x6f, 0x5f, 0x75, 0x73, 0x65,
	0x72, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x74, 0x6f,
	0x55, 0x73, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x72, 0x6f, 0x6f, 0x6d,
	0x5f, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x72, 0x6f, 0x6f, 0x6d, 0x49,
	0x64, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6d, 0x73, 0x67, 0x12, 0x0e, 0x0a, 0x02, 0x6f, 0x70, 0x18, 0x07, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x02, 0x6f, 0x70, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69,
	0x6d, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x54, 0x69, 0x6d, 0x65, 0x22, 0x22, 0x0a, 0x0c, 0x53, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x52,
	0x65, 0x70, 0x6c, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x32, 0x80, 0x05, 0x0a, 0x10, 0x43, 0x68, 0x61,
	0x74, 0x4c, 0x6f, 0x67, 0x69, 0x63, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x44, 0x0a,
	0x08, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x12, 0x1b, 0x2e, 0x6c, 0x6f, 0x67, 0x69,
	0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x70, 0x6c,
	0x79, 0x22, 0x00, 0x12, 0x3e, 0x0a, 0x05, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x12, 0x18, 0x2e, 0x6c,
	0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x00, 0x12, 0x4a, 0x0a, 0x09, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x41, 0x75, 0x74, 0x68,
	0x12, 0x1c, 0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x43, 0x68,
	0x65, 0x63, 0x6b, 0x41, 0x75, 0x74, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1d,
	0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x43, 0x68, 0x65, 0x63,
	0x6b, 0x41, 0x75, 0x74, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12,
	0x41, 0x0a, 0x06, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x12, 0x19, 0x2e, 0x6c, 0x6f, 0x67, 0x69,
	0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x1a, 0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x00, 0x12, 0x58, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x49, 0x6e, 0x66,
	0x6f, 0x42, 0x79, 0x55, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1e, 0x2e, 0x6c, 0x6f, 0x67, 0x69,
	0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x49, 0x6e,
	0x66, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1f, 0x2e, 0x6c, 0x6f, 0x67, 0x69,
	0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x49, 0x6e,
	0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x41, 0x0a, 0x07,
	0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x12, 0x1a, 0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x12,
	0x4a, 0x0a, 0x0a, 0x44, 0x69, 0x73, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x12, 0x1d, 0x2e,
	0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x44, 0x69, 0x73, 0x43, 0x6f,
	0x6e, 0x6e, 0x65, 0x63, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1b, 0x2e, 0x6c,
	0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x44, 0x69, 0x73, 0x43, 0x6f, 0x6e,
	0x6e, 0x65, 0x63, 0x74, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x12, 0x34, 0x0a, 0x04, 0x50,
	0x75, 0x73, 0x68, 0x12, 0x10, 0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x53, 0x65, 0x6e, 0x64, 0x1a, 0x18, 0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x53, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22,
	0x00, 0x12, 0x38, 0x0a, 0x08, 0x50, 0x75, 0x73, 0x68, 0x52, 0x6f, 0x6f, 0x6d, 0x12, 0x10, 0x2e,
	0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x53, 0x65, 0x6e, 0x64, 0x1a,
	0x18, 0x2e, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x53, 0x75, 0x63,
	0x63, 0x65, 0x73, 0x73, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x42, 0x0e, 0x5a, 0x0c, 0x2e,
	0x2f, 0x6c, 0x6f, 0x67, 0x69, 0x63, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
})

var (
	file_logic_proto_rawDescOnce sync.Once
	file_logic_proto_rawDescData []byte
)

func file_logic_proto_rawDescGZIP() []byte {
	file_logic_proto_rawDescOnce.Do(func() {
		file_logic_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_logic_proto_rawDesc), len(file_logic_proto_rawDesc)))
	})
	return file_logic_proto_rawDescData
}

var file_logic_proto_msgTypes = make([]protoimpl.MessageInfo, 16)
var file_logic_proto_goTypes = []any{
	(*RegisterRequest)(nil),     // 0: logicproto.RegisterRequest
	(*RegisterReply)(nil),       // 1: logicproto.RegisterReply
	(*LoginRequest)(nil),        // 2: logicproto.LoginRequest
	(*LoginResponse)(nil),       // 3: logicproto.LoginResponse
	(*CheckAuthRequest)(nil),    // 4: logicproto.CheckAuthRequest
	(*CheckAuthResponse)(nil),   // 5: logicproto.CheckAuthResponse
	(*LogoutRequest)(nil),       // 6: logicproto.LogoutRequest
	(*LogoutResponse)(nil),      // 7: logicproto.LogoutResponse
	(*GetUserInfoRequest)(nil),  // 8: logicproto.GetUserInfoRequest
	(*GetUserInfoResponse)(nil), // 9: logicproto.GetUserInfoResponse
	(*ConnectRequest)(nil),      // 10: logicproto.ConnectRequest
	(*ConnectReply)(nil),        // 11: logicproto.ConnectReply
	(*DisConnectRequest)(nil),   // 12: logicproto.DisConnectRequest
	(*DisConnectReply)(nil),     // 13: logicproto.DisConnectReply
	(*Send)(nil),                // 14: logicproto.Send
	(*SuccessReply)(nil),        // 15: logicproto.SuccessReply
}
var file_logic_proto_depIdxs = []int32{
	0,  // 0: logicproto.ChatLogicService.Register:input_type -> logicproto.RegisterRequest
	2,  // 1: logicproto.ChatLogicService.Login:input_type -> logicproto.LoginRequest
	4,  // 2: logicproto.ChatLogicService.CheckAuth:input_type -> logicproto.CheckAuthRequest
	6,  // 3: logicproto.ChatLogicService.Logout:input_type -> logicproto.LogoutRequest
	8,  // 4: logicproto.ChatLogicService.GetUserInfoByUserId:input_type -> logicproto.GetUserInfoRequest
	10, // 5: logicproto.ChatLogicService.Connect:input_type -> logicproto.ConnectRequest
	12, // 6: logicproto.ChatLogicService.DisConnect:input_type -> logicproto.DisConnectRequest
	14, // 7: logicproto.ChatLogicService.Push:input_type -> logicproto.Send
	14, // 8: logicproto.ChatLogicService.PushRoom:input_type -> logicproto.Send
	1,  // 9: logicproto.ChatLogicService.Register:output_type -> logicproto.RegisterReply
	3,  // 10: logicproto.ChatLogicService.Login:output_type -> logicproto.LoginResponse
	5,  // 11: logicproto.ChatLogicService.CheckAuth:output_type -> logicproto.CheckAuthResponse
	7,  // 12: logicproto.ChatLogicService.Logout:output_type -> logicproto.LogoutResponse
	9,  // 13: logicproto.ChatLogicService.GetUserInfoByUserId:output_type -> logicproto.GetUserInfoResponse
	11, // 14: logicproto.ChatLogicService.Connect:output_type -> logicproto.ConnectReply
	13, // 15: logicproto.ChatLogicService.DisConnect:output_type -> logicproto.DisConnectReply
	15, // 16: logicproto.ChatLogicService.Push:output_type -> logicproto.SuccessReply
	15, // 17: logicproto.ChatLogicService.PushRoom:output_type -> logicproto.SuccessReply
	9,  // [9:18] is the sub-list for method output_type
	0,  // [0:9] is the sub-list for method input_type
	0,  // [0:0] is the sub-list for extension type_name
	0,  // [0:0] is the sub-list for extension extendee
	0,  // [0:0] is the sub-list for field type_name
}

func init() { file_logic_proto_init() }
func file_logic_proto_init() {
	if File_logic_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_logic_proto_rawDesc), len(file_logic_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   16,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_logic_proto_goTypes,
		DependencyIndexes: file_logic_proto_depIdxs,
		MessageInfos:      file_logic_proto_msgTypes,
	}.Build()
	File_logic_proto = out.File
	file_logic_proto_goTypes = nil
	file_logic_proto_depIdxs = nil
}
