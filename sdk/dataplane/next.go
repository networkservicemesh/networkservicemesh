package dataplane

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/api/dataplane"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/utils/typeutils"
)

type next struct {
	handlers []dataplane.DataplaneServer
	index    int
}

func (n *next) Request(ctx context.Context, request *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	if n.index+1 < len(n.handlers) {
		ctx = withNext(ctx, &next{handlers: n.handlers, index: n.index + 1})
	} else {
		ctx = withNext(ctx, nil)
	}
	var span opentracing.Span
	logger := common.LogFromSpan(span)
	ctx = WithLogger(ctx, logger)
	logger.Infof("internal request %v", request)
	rv, err := n.handlers[n.index].Request(ctx, request)
	if err != nil {
		logger.Errorf("Error: %v", err)
		return nil, err
	}
	logger.Infof("internal response %v", rv)
	return rv, err
}

func (n *next) Close(ctx context.Context, request *crossconnect.CrossConnect) (*empty.Empty, error) {
	if n.index+1 < len(n.handlers) {
		ctx = withNext(ctx, &next{handlers: n.handlers, index: n.index + 1})
	} else {
		ctx = withNext(ctx, nil)
	}
	var span opentracing.Span
	if opentracing.IsGlobalTracerRegistered() {
		span, ctx = opentracing.StartSpanFromContext(ctx, fmt.Sprintf("%s.Close", typeutils.GetTypeName(n.handlers[n.index])))
		defer span.Finish()
	}
	logger := common.LogFromSpan(span)
	ctx = WithLogger(ctx, logger)
	logger.Infof("internal request %v", request)
	rv, err := n.handlers[n.index].Close(ctx, request)
	if err != nil {
		if span != nil {
			span.LogFields(log.Error(err))
		}
		logger.Error(err)
		return nil, err
	}
	logger.Infof("internal response %v", rv)
	return rv, err
}
