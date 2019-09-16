package tools

import (
	"context"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/mesos/mesos-go/api/v0/examples/Godeps/_workspace/src/github.com/gogo/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

func OpenTracingClientInterceptorWithClone(tracer opentracing.Tracer, optFuncs ...otgrpc.Option) grpc.UnaryClientInterceptor {
	uci := otgrpc.OpenTracingClientInterceptor(tracer, optFuncs...)

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return uci(ctx, method, proto.Clone(req.(proto.Message)), proto.Clone(reply.(proto.Message)), cc, invoker, opts...)
	}
}

func OpenTracingServerInterceptorWithClone(tracer opentracing.Tracer, optFuncs ...otgrpc.Option) grpc.UnaryServerInterceptor {
	usi := otgrpc.OpenTracingServerInterceptor(tracer, optFuncs...)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		return usi(ctx, proto.Clone(req.(proto.Message)), info, handler)
	}
}
