package protov1

import (
	"context"

	"google.golang.org/grpc"
)

// NodeControllerServiceClient is the client API for NodeControllerService service.
type NodeControllerServiceClient interface {
	EnrollNode(ctx context.Context, in *EnrollmentRequest, opts ...grpc.CallOption) (*EnrollmentResponse, error)
	RegisterNode(ctx context.Context, in *RegistrationRequest, opts ...grpc.CallOption) (*RegistrationResponse, error)
	SendHeartbeat(ctx context.Context, in *HeartbeatRequest, opts ...grpc.CallOption) (*HeartbeatResponse, error)
	SendTelemetry(ctx context.Context, in *TelemetryRequest, opts ...grpc.CallOption) (*TelemetryResponse, error)
	StreamControlChannel(ctx context.Context, opts ...grpc.CallOption) (NodeControllerService_StreamControlChannelClient, error)
}

type nodeControllerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewNodeControllerServiceClient(cc grpc.ClientConnInterface) NodeControllerServiceClient {
	return &nodeControllerServiceClient{cc}
}

func (c *nodeControllerServiceClient) EnrollNode(ctx context.Context, in *EnrollmentRequest, opts ...grpc.CallOption) (*EnrollmentResponse, error) {
	out := new(EnrollmentResponse)
	err := c.cc.Invoke(ctx, "/node_controller.v1.NodeControllerService/EnrollNode", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeControllerServiceClient) RegisterNode(ctx context.Context, in *RegistrationRequest, opts ...grpc.CallOption) (*RegistrationResponse, error) {
	out := new(RegistrationResponse)
	err := c.cc.Invoke(ctx, "/node_controller.v1.NodeControllerService/RegisterNode", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeControllerServiceClient) SendHeartbeat(ctx context.Context, in *HeartbeatRequest, opts ...grpc.CallOption) (*HeartbeatResponse, error) {
	out := new(HeartbeatResponse)
	err := c.cc.Invoke(ctx, "/node_controller.v1.NodeControllerService/SendHeartbeat", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeControllerServiceClient) SendTelemetry(ctx context.Context, in *TelemetryRequest, opts ...grpc.CallOption) (*TelemetryResponse, error) {
	out := new(TelemetryResponse)
	err := c.cc.Invoke(ctx, "/node_controller.v1.NodeControllerService/SendTelemetry", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type NodeControllerService_StreamControlChannelClient interface {
	Send(*NodeMessage) error
	Recv() (*ControllerMessage, error)
	grpc.ClientStream
}

type nodeControllerServiceStreamControlChannelClient struct {
	grpc.ClientStream
}

func (x *nodeControllerServiceStreamControlChannelClient) Send(m *NodeMessage) error {
	return x.ClientStream.SendMsg(m)
}

func (x *nodeControllerServiceStreamControlChannelClient) Recv() (*ControllerMessage, error) {
	m := new(ControllerMessage)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *nodeControllerServiceClient) StreamControlChannel(ctx context.Context, opts ...grpc.CallOption) (NodeControllerService_StreamControlChannelClient, error) {
	stream, err := c.cc.NewStream(ctx, &NodeControllerService_ServiceDesc.Streams[0], "/node_controller.v1.NodeControllerService/StreamControlChannel", opts...)
	if err != nil {
		return nil, err
	}
	x := &nodeControllerServiceStreamControlChannelClient{stream}
	return x, nil
}

// NodeControllerServiceServer is the server API for NodeControllerService service.
type NodeControllerServiceServer interface {
	EnrollNode(context.Context, *EnrollmentRequest) (*EnrollmentResponse, error)
	RegisterNode(context.Context, *RegistrationRequest) (*RegistrationResponse, error)
	SendHeartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error)
	SendTelemetry(context.Context, *TelemetryRequest) (*TelemetryResponse, error)
	StreamControlChannel(NodeControllerService_StreamControlChannelServer) error
}

type UnimplementedNodeControllerServiceServer struct{}

func (UnimplementedNodeControllerServiceServer) EnrollNode(context.Context, *EnrollmentRequest) (*EnrollmentResponse, error) {
	return nil, nil
}
func (UnimplementedNodeControllerServiceServer) RegisterNode(context.Context, *RegistrationRequest) (*RegistrationResponse, error) {
	return nil, nil
}
func (UnimplementedNodeControllerServiceServer) SendHeartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error) {
	return nil, nil
}
func (UnimplementedNodeControllerServiceServer) SendTelemetry(context.Context, *TelemetryRequest) (*TelemetryResponse, error) {
	return nil, nil
}
func (UnimplementedNodeControllerServiceServer) StreamControlChannel(NodeControllerService_StreamControlChannelServer) error {
	return nil
}

func RegisterNodeControllerServiceServer(s grpc.ServiceRegistrar, srv NodeControllerServiceServer) {
	s.RegisterService(&NodeControllerService_ServiceDesc, srv)
}

type NodeControllerService_StreamControlChannelServer interface {
	Send(*ControllerMessage) error
	Recv() (*NodeMessage, error)
	grpc.ServerStream
}

type nodeControllerServiceStreamControlChannelServer struct {
	grpc.ServerStream
}

func (x *nodeControllerServiceStreamControlChannelServer) Send(m *ControllerMessage) error {
	return x.ServerStream.SendMsg(m)
}

func (x *nodeControllerServiceStreamControlChannelServer) Recv() (*NodeMessage, error) {
	m := new(NodeMessage)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _NodeControllerService_EnrollNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EnrollmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeControllerServiceServer).EnrollNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/node_controller.v1.NodeControllerService/EnrollNode",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeControllerServiceServer).EnrollNode(ctx, req.(*EnrollmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeControllerService_RegisterNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegistrationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeControllerServiceServer).RegisterNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/node_controller.v1.NodeControllerService/RegisterNode",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeControllerServiceServer).RegisterNode(ctx, req.(*RegistrationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeControllerService_SendHeartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HeartbeatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeControllerServiceServer).SendHeartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/node_controller.v1.NodeControllerService/SendHeartbeat",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeControllerServiceServer).SendHeartbeat(ctx, req.(*HeartbeatRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeControllerService_SendTelemetry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TelemetryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeControllerServiceServer).SendTelemetry(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/node_controller.v1.NodeControllerService/SendTelemetry",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeControllerServiceServer).SendTelemetry(ctx, req.(*TelemetryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeControllerService_StreamControlChannel_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(NodeControllerServiceServer).StreamControlChannel(&nodeControllerServiceStreamControlChannelServer{stream})
}

var NodeControllerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "node_controller.v1.NodeControllerService",
	HandlerType: (*NodeControllerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "EnrollNode",
			Handler:    _NodeControllerService_EnrollNode_Handler,
		},
		{
			MethodName: "RegisterNode",
			Handler:    _NodeControllerService_RegisterNode_Handler,
		},
		{
			MethodName: "SendHeartbeat",
			Handler:    _NodeControllerService_SendHeartbeat_Handler,
		},
		{
			MethodName: "SendTelemetry",
			Handler:    _NodeControllerService_SendTelemetry_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamControlChannel",
			Handler:       _NodeControllerService_StreamControlChannel_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "api/proto/node_controller.proto",
}
