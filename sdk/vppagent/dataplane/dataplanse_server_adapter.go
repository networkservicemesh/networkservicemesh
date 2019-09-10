package dataplane

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
)

type adapter struct {
	request func(context.Context, *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error)
	close   func(context.Context, *crossconnect.CrossConnect) (*empty.Empty, error)
}

func (a *adapter) Request(ctx context.Context, conn *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	if a.request == nil {
		return conn, nil
	}
	return a.request(ctx, conn)
}

func (a *adapter) Close(ctx context.Context, conn *crossconnect.CrossConnect) (*empty.Empty, error) {
	if a.close == nil {
		return new(empty.Empty), nil
	}
	return a.close(ctx, conn)
}

func AdaptCloseFunc(close func(context.Context, *crossconnect.CrossConnect) (*empty.Empty, error)) dataplane.DataplaneServer {
	return &adapter{
		request: nil,
		close:   close,
	}
}

func AdaptRequestFunc(request func(context.Context, *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error)) dataplane.DataplaneServer {
	return &adapter{
		request: request,
		close:   nil,
	}
}
