package refresh

import (
	"context"
	"runtime"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/tools/extended_context"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/tools/serialize"
)

const (
	defaultBufferSize = 100
)

type timeoutClient struct {
	connectionTimers map[string]*time.Timer
	connections      map[string]*connection.Connection
	executor         serialize.Executor
}

func NewClient() networkservice.NetworkServiceClient {
	rv := &timeoutClient{
		connectionTimers: make(map[string]*time.Timer),
		connections:      make(map[string]*connection.Connection),
		executor:         serialize.NewExecutor(),
	}
	runtime.SetFinalizer(rv, func(client *timeoutClient) {
		for _, conn := range client.connections {
			// TODO look into non-background context here
			// TODO look into wrapping a span around this call perhaps?
			rv.Close(context.Background(), conn)
		}
	})
	return rv
}

func (t *timeoutClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	rv, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling next")
	}
	// Clone the request
	req := request.Clone()
	// Set its connection to the returned connection we received
	req.Connection = rv
	// Setup the timer with the req containing the returned connection
	ct, err := t.createTimer(ctx, req, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating timer from Request.Connection.Path.PathSegment[%d].ExpireTime", request.GetConnection().GetPath().GetIndex())
	}
	t.executor.AsyncExec(func() {
		if timer, ok := t.connectionTimers[request.GetConnection().GetId()]; !ok || timer.Stop() {
			t.connectionTimers[request.GetConnection().GetId()] = ct
		}
	})
	return rv, nil
}

func (t *timeoutClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	t.executor.AsyncExec(func() {
		if timer, ok := t.connectionTimers[conn.GetId()]; ok {
			timer.Stop()
			delete(t.connectionTimers, conn.GetId())
		}
	})
	return next.Server(ctx).Close(ctx, conn)
}

func (t *timeoutClient) createTimer(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*time.Timer, error) {
	expireTime, err := ptypes.Timestamp(request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].GetExpires())
	if err != nil {
		return nil, err
	}
	duration := expireTime.Sub(time.Now())
	if err != nil {
		return nil, err
	}

	return time.AfterFunc(duration, func() {
		// TODO what to do about error handling?
		// TODO what to do about expiration of context
		t.Request(extended_context.New(context.Background(), ctx), request, opts...)
	}), nil

}
