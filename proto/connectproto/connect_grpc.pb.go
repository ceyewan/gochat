// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v4.25.3
// source: connect.proto

package connectproto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	ConnectService_PushSingleMsg_FullMethodName = "/connectproto.ConnectService/PushSingleMsg"
	ConnectService_PushRoomMsg_FullMethodName   = "/connectproto.ConnectService/PushRoomMsg"
	ConnectService_PushRoomInfo_FullMethodName  = "/connectproto.ConnectService/PushRoomInfo"
)

// ConnectServiceClient is the client API for ConnectService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// 消息服务定义
type ConnectServiceClient interface {
	// 推送单个消息
	PushSingleMsg(ctx context.Context, in *PushSingleMsgRequest, opts ...grpc.CallOption) (*SuccessReply, error)
	// 推送房间消息
	PushRoomMsg(ctx context.Context, in *PushRoomMsgRequest, opts ...grpc.CallOption) (*SuccessReply, error)
	// 推送房间信息
	PushRoomInfo(ctx context.Context, in *PushRoomInfoRequest, opts ...grpc.CallOption) (*SuccessReply, error)
}

type connectServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewConnectServiceClient(cc grpc.ClientConnInterface) ConnectServiceClient {
	return &connectServiceClient{cc}
}

func (c *connectServiceClient) PushSingleMsg(ctx context.Context, in *PushSingleMsgRequest, opts ...grpc.CallOption) (*SuccessReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SuccessReply)
	err := c.cc.Invoke(ctx, ConnectService_PushSingleMsg_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *connectServiceClient) PushRoomMsg(ctx context.Context, in *PushRoomMsgRequest, opts ...grpc.CallOption) (*SuccessReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SuccessReply)
	err := c.cc.Invoke(ctx, ConnectService_PushRoomMsg_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *connectServiceClient) PushRoomInfo(ctx context.Context, in *PushRoomInfoRequest, opts ...grpc.CallOption) (*SuccessReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SuccessReply)
	err := c.cc.Invoke(ctx, ConnectService_PushRoomInfo_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ConnectServiceServer is the server API for ConnectService service.
// All implementations must embed UnimplementedConnectServiceServer
// for forward compatibility.
//
// 消息服务定义
type ConnectServiceServer interface {
	// 推送单个消息
	PushSingleMsg(context.Context, *PushSingleMsgRequest) (*SuccessReply, error)
	// 推送房间消息
	PushRoomMsg(context.Context, *PushRoomMsgRequest) (*SuccessReply, error)
	// 推送房间信息
	PushRoomInfo(context.Context, *PushRoomInfoRequest) (*SuccessReply, error)
	mustEmbedUnimplementedConnectServiceServer()
}

// UnimplementedConnectServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedConnectServiceServer struct{}

func (UnimplementedConnectServiceServer) PushSingleMsg(context.Context, *PushSingleMsgRequest) (*SuccessReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PushSingleMsg not implemented")
}
func (UnimplementedConnectServiceServer) PushRoomMsg(context.Context, *PushRoomMsgRequest) (*SuccessReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PushRoomMsg not implemented")
}
func (UnimplementedConnectServiceServer) PushRoomInfo(context.Context, *PushRoomInfoRequest) (*SuccessReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PushRoomInfo not implemented")
}
func (UnimplementedConnectServiceServer) mustEmbedUnimplementedConnectServiceServer() {}
func (UnimplementedConnectServiceServer) testEmbeddedByValue()                        {}

// UnsafeConnectServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ConnectServiceServer will
// result in compilation errors.
type UnsafeConnectServiceServer interface {
	mustEmbedUnimplementedConnectServiceServer()
}

func RegisterConnectServiceServer(s grpc.ServiceRegistrar, srv ConnectServiceServer) {
	// If the following call pancis, it indicates UnimplementedConnectServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ConnectService_ServiceDesc, srv)
}

func _ConnectService_PushSingleMsg_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PushSingleMsgRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConnectServiceServer).PushSingleMsg(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ConnectService_PushSingleMsg_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConnectServiceServer).PushSingleMsg(ctx, req.(*PushSingleMsgRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ConnectService_PushRoomMsg_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PushRoomMsgRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConnectServiceServer).PushRoomMsg(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ConnectService_PushRoomMsg_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConnectServiceServer).PushRoomMsg(ctx, req.(*PushRoomMsgRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ConnectService_PushRoomInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PushRoomInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConnectServiceServer).PushRoomInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ConnectService_PushRoomInfo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConnectServiceServer).PushRoomInfo(ctx, req.(*PushRoomInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ConnectService_ServiceDesc is the grpc.ServiceDesc for ConnectService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ConnectService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "connectproto.ConnectService",
	HandlerType: (*ConnectServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PushSingleMsg",
			Handler:    _ConnectService_PushSingleMsg_Handler,
		},
		{
			MethodName: "PushRoomMsg",
			Handler:    _ConnectService_PushRoomMsg_Handler,
		},
		{
			MethodName: "PushRoomInfo",
			Handler:    _ConnectService_PushRoomInfo_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "connect.proto",
}
