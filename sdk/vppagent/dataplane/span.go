package dataplane

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

type span struct {
}

func (c *span) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	var span opentracing.Span
	var nextContext context.Context
	if opentracing.GlobalTracer() != nil {
		span, nextContext = opentracing.StartSpanFromContext(ctx, "DataplaneServer.Request")
		defer span.Finish()
	}
	nextContext = context.WithValue(nextContext, loggerKey, common.LogFromSpan(span))
	if next := Next(nextContext); next != nil {
		return next.Request(nextContext, crossConnect)
	}
	return crossConnect, nil
}

func (c *span) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	var span opentracing.Span
	var nextContext context.Context
	if opentracing.GlobalTracer() != nil {
		span, nextContext = opentracing.StartSpanFromContext(ctx, "DataplaneServer.Close")
		defer span.Finish()
	}
	if next := Next(nextContext); next != nil {
		return next.Close(nextContext, crossConnect)
	}
	return new(empty.Empty), nil
}

//UseSpan creates dataplane server handler with span injection
func UseSpan() dataplane.DataplaneServer {
	return &span{}
}
