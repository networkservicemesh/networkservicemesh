package forwarder

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/api/dataplane"
)

//ConnectionValidator returns Dataplane Server with validation for Request and Close
func ConnectionValidator() dataplane.DataplaneServer {
	return &validator{}
}

type validator struct {
}

func (v *validator) Request(ctx context.Context, request *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	if err := request.IsValid(); err != nil {
		Logger(ctx).Errorf("Reuqest: %v is not valid, reason: %v", request, err)
		return nil, err
	}
	if next := Next(ctx); next != nil {
		return next.Request(ctx, request)
	}
	return request, nil
}

func (v *validator) Available(ctx context.Context, list *dataplane.CrossConnectList) (*dataplane.CrossConnectList, error) {
	if next := Next(ctx); next != nil {
		return next.Available(ctx, list)
	}
	return list, nil
}

func (v *validator) Close(ctx context.Context, request *crossconnect.CrossConnect) (*empty.Empty, error) {
	if err := request.IsValid(); err != nil {
		Logger(ctx).Errorf("Close: %v is not valid, reason: %v", request, err)
		return new(empty.Empty), err
	}
	if next := Next(ctx); next != nil {
		return next.Close(ctx, request)
	}
	return new(empty.Empty), nil
}
