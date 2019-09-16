package tools

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"reflect"
)

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

func CloneArgsClientInterceptor(uci grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
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

func CloneArgsServerInterceptor(usi grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		return usi(ctx, proto.Clone(req.(proto.Message)), info, handler)
	}
}
