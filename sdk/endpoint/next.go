package endpoint

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/utils/typeutils"
)

type nextEndpoint struct {
	composite *CompositeEndpoint
	index     int
}

func (n *nextEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if n.index+1 < len(n.composite.endpoints) {
		ctx = withNext(ctx, &nextEndpoint{composite: n.composite, index: n.index + 1})
	} else {
		ctx = withNext(ctx, nil)
	}

	// Create a new span
	var span opentracing.Span
	if opentracing.IsGlobalTracerRegistered() {
		span, ctx = opentracing.StartSpanFromContext(ctx, fmt.Sprintf("%s.Request", typeutils.GetTypeName(n.composite.endpoints[n.index])))
		defer span.Finish()

		// Make sure we log to span
	}
	logger := common.LogFromSpan(span)

	ctx = withLog(ctx, logger)
	logger.Infof("internal request %v", request)

	// Actually call the next
	rv, err := n.composite.endpoints[n.index].Request(ctx, request)

	if err != nil {
		logger.Errorf("Error: %v", err)
		return nil, err
	}
	logger.Infof("internal response %v", rv)
	return rv, err
}

func (n *nextEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if n.index+1 < len(n.composite.endpoints) {
		ctx = withNext(ctx, &nextEndpoint{composite: n.composite, index: n.index + 1})
	} else {
		ctx = withNext(ctx, nil)
	}
	// Create a new span
	var span opentracing.Span
	if opentracing.IsGlobalTracerRegistered() {
		span, ctx = opentracing.StartSpanFromContext(ctx, fmt.Sprintf("%s.Close", typeutils.GetTypeName(n.composite.endpoints[n.index])))
		defer span.Finish()
	}
	// Make sure we log to span
	logger := common.LogFromSpan(span)
	ctx = withLog(ctx, logger)

	logger.Infof("internal request %v", connection)
	rv, err := n.composite.endpoints[n.index].Close(ctx, connection)

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
