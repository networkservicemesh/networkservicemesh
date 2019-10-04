package tools

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
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
		span := spanhelper.GetSpanHelper(ctx)
		replyPtr := allocate(dereferenceType(reply))
		reqCopy := proto.Clone(req.(proto.Message))
		span.LogObject(fmt.Sprintf("%v()", method), reqCopy )
		err := uci(ctx, method, reqCopy, replyPtr, cc, invoker, opts...)
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
