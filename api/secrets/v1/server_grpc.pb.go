// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v5.28.0
// source: api/secrets/v1/server.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// SecretsClient is the client API for Secrets service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SecretsClient interface {
	FetchSecret(ctx context.Context, in *FetchSecretRequest, opts ...grpc.CallOption) (*FetchSecretResponse, error)
}

type secretsClient struct {
	cc grpc.ClientConnInterface
}

func NewSecretsClient(cc grpc.ClientConnInterface) SecretsClient {
	return &secretsClient{cc}
}

func (c *secretsClient) FetchSecret(ctx context.Context, in *FetchSecretRequest, opts ...grpc.CallOption) (*FetchSecretResponse, error) {
	out := new(FetchSecretResponse)
	err := c.cc.Invoke(ctx, "/secrets.Secrets/FetchSecret", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SecretsServer is the server API for Secrets service.
// All implementations must embed UnimplementedSecretsServer
// for forward compatibility
type SecretsServer interface {
	FetchSecret(context.Context, *FetchSecretRequest) (*FetchSecretResponse, error)
	mustEmbedUnimplementedSecretsServer()
}

// UnimplementedSecretsServer must be embedded to have forward compatible implementations.
type UnimplementedSecretsServer struct {
}

func (UnimplementedSecretsServer) FetchSecret(context.Context, *FetchSecretRequest) (*FetchSecretResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchSecret not implemented")
}
func (UnimplementedSecretsServer) mustEmbedUnimplementedSecretsServer() {}

// UnsafeSecretsServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SecretsServer will
// result in compilation errors.
type UnsafeSecretsServer interface {
	mustEmbedUnimplementedSecretsServer()
}

func RegisterSecretsServer(s grpc.ServiceRegistrar, srv SecretsServer) {
	s.RegisterService(&Secrets_ServiceDesc, srv)
}

func _Secrets_FetchSecret_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchSecretRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SecretsServer).FetchSecret(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/secrets.Secrets/FetchSecret",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SecretsServer).FetchSecret(ctx, req.(*FetchSecretRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Secrets_ServiceDesc is the grpc.ServiceDesc for Secrets service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Secrets_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "secrets.Secrets",
	HandlerType: (*SecretsServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "FetchSecret",
			Handler:    _Secrets_FetchSecret_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/secrets/v1/server.proto",
}
