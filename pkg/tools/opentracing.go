package tools

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"reflect"
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
		replyPtr := allocate(dereferenceType(reply))
		err := uci(ctx, method, proto.Clone(req.(proto.Message)), replyPtr, cc, invoker, opts...)
		memset(reply, replyPtr.(proto.Message))
		return err
	}
}

func allocate(typ reflect.Type) interface{} {
	return reflect.New(typ).Interface()
}

func dereferenceType(ptr interface{}) reflect.Type {
	return reflect.Indirect(reflect.ValueOf(ptr)).Type()
}

func memset(ptr interface{}, value proto.Message) {
	clone := reflect.ValueOf(proto.Clone(value)).Elem()
	reflect.ValueOf(ptr).Elem().Set(clone)
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
