package compat

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
)

type networkServiceServerAdapter struct {
	remoteServer remote.NetworkServiceServer
	localServer  local.NetworkServiceServer
}

func (s networkServiceServerAdapter) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
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
			return nil, errors.Errorf("networkServiceServerAdapter.Request() received mixed local and remote request.GetMechanismPreferences(): %+v", request.GetMechanismPreferences())
		}
	}
	if class == cls.LOCAL && s.localServer != nil {
		conn, err := s.localServer.Request(ctx, NetworkServiceRequestUnifiedToLocal(request))
		if err != nil {
			return nil, err
		}
		return ConnectionLocalToUnified(conn), nil
	} else if class == cls.REMOTE && s.remoteServer != nil {
		conn, err := s.remoteServer.Request(ctx, NetworkServiceRequestUnifiedToRemote(request))
		if err != nil {
			return nil, err
		}
		return ConnectionRemoteToUnified(conn), nil
	}
	return nil, errors.Errorf("No NetworkServiceRequestServer available for request.GetMechanismPreferences() Mechanism Cls: %s", class)
}

func (s networkServiceServerAdapter) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if err := conn.IsValid(); err != nil {
		return nil, err
	}

	if conn.GetMechanism().GetCls() == cls.LOCAL && s.localServer != nil {
		return s.localServer.Close(ctx, ConnectionUnifiedToLocal(conn))
	} else if conn.GetMechanism().GetCls() == cls.REMOTE && s.remoteServer != nil {
		return s.remoteServer.Close(ctx, ConnectionUnifiedToRemote(conn))
	}

	return nil, errors.Errorf("No NetworkServiceServer available for Connection.GetMechanism().GetCls(): %+s", conn.GetMechanism().GetCls())
}

func NewUnifiedNetworkServiceServerAdapter(remoteServer remote.NetworkServiceServer, localServer local.NetworkServiceServer) networkservice.NetworkServiceServer {
	return &networkServiceServerAdapter{
		remoteServer: remoteServer,
		localServer:  localServer,
	}
}
