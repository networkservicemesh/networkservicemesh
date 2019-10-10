package forwarder

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/api/dataplane"
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
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.Request", typeutils.GetTypeName(n.handlers[n.index])))
	defer span.Finish()
	ctx = WithLogger(span.Context(), span.Logger())
	span.LogObject("request", request)
	rv, err := n.handlers[n.index].Request(ctx, request)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", rv)
	return rv, err
}

func (n *next) Close(ctx context.Context, request *crossconnect.CrossConnect) (*empty.Empty, error) {
	if n.index+1 < len(n.handlers) {
		ctx = withNext(ctx, &next{handlers: n.handlers, index: n.index + 1})
	} else {
		ctx = withNext(ctx, nil)
	}
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.Close", typeutils.GetTypeName(n.handlers[n.index])))
	defer span.Finish()
	ctx = WithLogger(span.Context(), span.Logger())
	span.LogObject("request", request)
	rv, err := n.handlers[n.index].Close(ctx, request)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", rv)
	return rv, err
}

func (n *next) Available(ctx context.Context, request *dataplane.CrossConnectList) (*dataplane.CrossConnectList, error) {
	if n.index+1 < len(n.handlers) {
		ctx = withNext(ctx, &next{handlers: n.handlers, index: n.index + 1})
	} else {
		ctx = withNext(ctx, nil)
	}
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.Available", typeutils.GetTypeName(n.handlers[n.index])))
	defer span.Finish()
	ctx = WithLogger(span.Context(), span.Logger())
	span.LogObject("request", request)
	rv, err := n.handlers[n.index].Available(ctx, request)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", rv)
	return rv, err
}
