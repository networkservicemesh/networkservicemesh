package endpoint

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/utils/typeutils"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
)

type nextEndpoint struct {
	composite *CompositeEndpoint
	index     int
}

func (n *nextEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = withNext(ctx, nil)
	if n.index+1 < len(n.composite.endpoints) {
		ctx = withNext(ctx, &nextEndpoint{composite: n.composite, index: n.index + 1})
	}

	// Create a new span
	span, ctx := opentracing.StartSpanFromContext(ctx, typeutils.GetTypeName(n.composite.endpoints[n.index]))
	defer span.Finish()

	// Make sure we log to span
	ctx = withLog(ctx, LogFromSpan(span))

	span.LogFields(log.Object("internal request", request))

	// Actually call the next
	rv, err := n.composite.endpoints[n.index].Request(ctx, request)

	if err != nil {
		span.LogFields(log.Error(err))
		logrus.Error(err)
		return nil, err
	}
	span.LogFields(log.Object("internal response", rv))
	return rv, err
}

func (n *nextEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	ctx = withNext(ctx, nil)
	if n.index < len(n.composite.endpoints) {
		ctx = withNext(ctx, &nextEndpoint{composite: n.composite, index: n.index + 1})
	}
	// Create a new span
	span, ctx := opentracing.StartSpanFromContext(ctx, typeutils.GetTypeName(n.composite.endpoints[n.index]))
	defer span.Finish()

	// Make sure we log to span
	ctx = withLog(ctx, LogFromSpan(span))

	span.LogFields(log.Object("internal request", connection))
	rv, err := n.composite.endpoints[n.index].Close(ctx, connection)

	if err != nil {
		span.LogFields(log.Error(err))
		logrus.Error(err)
		return nil, err
	}
	span.LogFields(log.Object("internal response", rv))
	return rv, err
}
