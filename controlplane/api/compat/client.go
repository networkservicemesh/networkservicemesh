package compat

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"google.golang.org/grpc"
)

type clientAdapter struct {
	localClient  local.NetworkServiceClient
	remoteClient remote.NetworkServiceClient
}

func NewClientAdapter(remoteClient remote.NetworkServiceClient, localClient local.NetworkServiceClient) networkservice.NetworkServiceClient {
	return &clientAdapter{
		localClient:  localClient,
		remoteClient: remoteClient,
	}
}

func (c clientAdapter) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	if err := request.IsValid(); err != nil {
		return nil, err
	}
	// Make sure all the MechanismPreferences are either local or remote... no mixed case
	class := ""
	for _, v := range request.GetMechanismPreferences() {
		if v.Cls == cls.LOCAL && (class == "" || class == cls.LOCAL) {
			class = cls.LOCAL
		} else if v.Cls == cls.REMOTE && (class == "" || class == cls.REMOTE) {
			class = cls.REMOTE
		} else {
			return nil, fmt.Errorf("serverAdapter.Request() received mixed local and remote request.GetMechanismPreferences(): %+v", request.GetMechanismPreferences())
		}
	}
	if class == cls.LOCAL && c.localClient != nil {
		conn, err := c.localClient.Request(ctx, NetworkServiceRequestUnifiedToLocal(request))
		if err != nil {
			return nil, err
		}
		return ConnectionLocalToUnified(conn), nil
	} else if class == cls.REMOTE && c.remoteClient != nil {
		conn, err := c.remoteClient.Request(ctx, NetworkServiceRequestUnifiedToRemote(request))
		if err != nil {
			return nil, err
		}
		return ConnectionRemoteToUnified(conn), nil
	}
	return nil, fmt.Errorf("No NetworkServiceRequestClient available for request.GetMechanismPreferences() Mechanism Cls: %s", class)
}

func (c clientAdapter) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if err := conn.IsValid(); err != nil {
		return nil, err
	}

	if conn.GetMechanism().GetCls() == cls.LOCAL && c.localClient != nil {
		return c.localClient.Close(ctx, ConnectionUnifiedToLocal(conn))
	} else if conn.GetMechanism().GetCls() == cls.REMOTE && c.localClient != nil {
		return c.remoteClient.Close(ctx, ConnectionUnifiedToRemote(conn))
	} else {
		return nil, fmt.Errorf("No NetworkServiceClient available for Connection.GetMechanism().GetCls(): %+s", conn.GetMechanism().GetCls())
	}

	return &empty.Empty{}, nil
}
