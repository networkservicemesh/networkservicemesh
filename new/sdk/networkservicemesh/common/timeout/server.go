package timeout

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/tools/extended_context"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/tools/serialize"
	"github.com/pkg/errors"
)

type timeoutServer struct {
	connections map[string]*time.Timer
	executor    serialize.Executor
}

func NewServer() networkservice.NetworkServiceServer {
	ts := &timeoutServer{
		connections: make(map[string]*time.Timer),
		executor:    serialize.NewExecutor(),
	}
	return ts
}

func (t *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ct, err := t.createTimer(ctx, request)
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating timer from Request.Connection.Path.PathSegment[%d].ExpireTime", request.GetConnection().GetPath().GetIndex())
	}
	t.executor.AsyncExec(func() {
		if timer, ok := t.connections[request.GetConnection().GetId()]; !ok || timer.Stop() {
			t.connections[request.GetConnection().GetId()] = ct
		}
	})
	return next.Server(ctx).Request(ctx, request)
}

func (t *timeoutServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	t.executor.AsyncExec(func() {
		if timer, ok := t.connections[conn.GetId()]; ok {
			timer.Stop()
			delete(t.connections, conn.GetId())
		}
	})
	return next.Server(ctx).Close(ctx, conn)
}

func (t *timeoutServer) createTimer(ctx context.Context, request *networkservice.NetworkServiceRequest) (*time.Timer, error) {
	expireTime, err := ptypes.Timestamp(request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].GetExpires())
	if err != nil {
		return nil, err
	}
	duration := expireTime.Sub(time.Now())
	if err != nil {
		return nil, err
	}
	return time.AfterFunc(duration, func() {
		t.Close(extended_context.New(context.Background(), ctx), request.GetConnection())
	}), nil
}
