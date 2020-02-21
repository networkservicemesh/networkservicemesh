package endpoint

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/utils/typeutils"
)

type nextEndpoint struct {
	composite *CompositeEndpoint
	index     int
}

func (n *nextEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if n.index+1 < len(n.composite.endpoints) {
		ctx = withNext(ctx, &nextEndpoint{composite: n.composite, index: n.index + 1})
	} else {
		ctx = withNext(ctx, nil)
	}

	// Create a new span
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.Request", typeutils.GetTypeName(n.composite.endpoints[n.index])))
	defer span.Finish()

	// Make sure we log to span

	ctx = withLog(span.Context(), span.Logger())

	span.LogObject("request", request)

	// Actually call the next
	rv, err := n.composite.endpoints[n.index].Request(ctx, request)

	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", rv)
	return rv, err
}

func (n *nextEndpoint) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	if n.index+1 < len(n.composite.endpoints) {
		ctx = withNext(ctx, &nextEndpoint{composite: n.composite, index: n.index + 1})
	} else {
		ctx = withNext(ctx, nil)
	}
	// Create a new span
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.Close", typeutils.GetTypeName(n.composite.endpoints[n.index])))
	defer span.Finish()
	// Make sure we log to span
	ctx = withLog(span.Context(), span.Logger())

	span.LogObject("request", connection)
	rv, err := n.composite.endpoints[n.index].Close(ctx, connection)

	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", rv)
	return rv, err
}
