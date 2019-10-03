package services

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

type ClientConnectionManager struct {
	model           model.Model
	serviceRegistry serviceregistry.ServiceRegistry
	manager         nsm.NetworkServiceManager
}

func NewClientConnectionManager(model model.Model, manager nsm.NetworkServiceManager, serviceRegistry serviceregistry.ServiceRegistry) *ClientConnectionManager {
	return &ClientConnectionManager{
		model:           model,
		serviceRegistry: serviceRegistry,
		manager:         manager,
	}
}

func (m *ClientConnectionManager) GetNsmName() string {
	return m.model.GetNsm().Name
}

// UpdateXcon handles case when xcon has been changed for NSMClientConnection
func (m *ClientConnectionManager) UpdateXcon(ctx context.Context, cc nsm.ClientConnection, newXcon *crossconnect.CrossConnect) {
	if upd := m.model.ApplyClientConnectionChanges(ctx, cc.GetID(), func(cc *model.ClientConnection) {
		cc.Xcon = newXcon
	}); upd != nil {
		cc = upd
	} else {
		logrus.Errorf("Trying to update not existing connection: %v", cc.GetID())
		return
	}

	if src := newXcon.GetLocalSource(); src != nil && src.State == local.State_DOWN {
		logrus.Info("ClientConnection src state is down")
		_ = m.manager.Close(context.Background(), cc)
		return
	}

	if dst := newXcon.GetLocalDestination(); dst != nil && dst.State == local.State_DOWN {
		logrus.Info("ClientConnection dst state is down")
		m.manager.Heal(cc, nsm.HealStateDstDown)
		return
	}
}

// DestinationDown handles case when destination down
func (m *ClientConnectionManager) DestinationDown(cc nsm.ClientConnection, nsmdDie bool) {
	if nsmdDie {
		m.manager.Heal(cc, nsm.HealStateDstNmgrDown)
	} else {
		m.manager.Heal(cc, nsm.HealStateDstDown)
	}
}

// DataplaneDown handles case of local dp down
func (m *ClientConnectionManager) DataplaneDown(dataplane *model.Dataplane) {
	ccs := m.model.GetAllClientConnections()

	for _, cc := range ccs {
		if cc.DataplaneRegisteredName == dataplane.RegisteredName {
			m.manager.Heal(cc, nsm.HealStateDataplaneDown)
		}
	}
}

// LocalDestinationUpdated handles case when local connection parameters changed
func (m *ClientConnectionManager) LocalDestinationUpdated(ctx context.Context, cc *model.ClientConnection, localDst *local.Connection) {
	if cc.ConnectionState != model.ClientConnectionReady {
		return
	}

	// NSE is not aware of 'Workspace' and 'WorkspaceNSEName' connection mechanism parameters
	localDst.GetMechanism().GetParameters()[local.Workspace] =
		cc.Xcon.GetLocalDestination().GetMechanism().GetParameters()[local.Workspace]
	localDst.GetMechanism().GetParameters()[local.WorkspaceNSEName] =
		cc.Xcon.GetLocalDestination().GetMechanism().GetParameters()[local.WorkspaceNSEName]

	m.destinationUpdated(ctx, cc, localDst)
}

// RemoteDestinationUpdated handles case when remote connection parameters changed
func (m *ClientConnectionManager) RemoteDestinationUpdated(ctx context.Context, cc *model.ClientConnection, remoteDst *remote.Connection) {
	if cc.ConnectionState != model.ClientConnectionReady {
		return
	}

	if remoteDst.State == remote.State_UP {
		// TODO: in order to update connection parameters we need to update model here
		// We do not need to heal in case DST state is UP, remote NSM will try to recover and only when will send Update, Delete of connection.
		return
	}

	m.destinationUpdated(ctx, cc, remoteDst)
}

func (m *ClientConnectionManager) destinationUpdated(ctx context.Context, cc nsm.ClientConnection, dst connection.Connection) {
	// Check if it update we already have
	if dst.Equals(cc.GetConnectionDestination()) {
		// Since they are same, we do not need to do anything.
		return
	}

	if upd := m.model.ApplyClientConnectionChanges(ctx, cc.GetID(), func(cc *model.ClientConnection) {
		cc.Xcon.SetDestinationConnection(dst)
	}); upd != nil {
		cc = upd
	} else {
		logrus.Errorf("Trying to update not existing connection: %v", cc.GetID())
		return
	}

	m.manager.Heal(cc, nsm.HealStateDstUpdate)
}

func (m *ClientConnectionManager) GetClientConnectionByXcon(xcon *crossconnect.CrossConnect) *model.ClientConnection {
	if dst := xcon.GetDestinationConnection(); dst.IsRemote() {
		return m.GetClientConnectionByRemoteDst(dst.GetId())
	} else {
		return m.GetClientConnectionByLocalDst(dst.GetId())
	}
}

// GetClientConnectionByLocalDst returns a ClientConnection with `Xcon.GetLocalDestination().GetID() == dstID`
// or `null` if there is no such connection
func (m *ClientConnectionManager) GetClientConnectionByLocalDst(dstID string) *model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()

	for _, clientConnection := range clientConnections {
		if dst := clientConnection.Xcon.GetLocalDestination(); dst != nil && dst.GetId() == dstID {
			return clientConnection
		}
	}

	return nil
}

// GetClientConnectionByRemoteDst returns a ClientConnection with `Xcon.GetRemoteDestination().GetId() == dstID`
// or `null` if there is no such connection
func (m *ClientConnectionManager) GetClientConnectionByRemoteDst(dstID string) *model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()

	for _, clientConnection := range clientConnections {
		if dst := clientConnection.Xcon.GetRemoteDestination(); dst != nil && dst.GetId() == dstID {
			return clientConnection
		}
	}

	return nil
}

func (m *ClientConnectionManager) GetClientConnectionByRemote(nsm *registry.NetworkServiceManager) []*model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()
	var result []*model.ClientConnection
	for _, clientConnection := range clientConnections {
		if clientConnection.RemoteNsm.GetName() == nsm.GetName() {
			result = append(result, clientConnection)
		}
	}
	return result
}

func (m *ClientConnectionManager) GetClientConnectionsByDataplane(name string) []*model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()

	var rv []*model.ClientConnection
	for _, clientConnection := range clientConnections {
		if clientConnection.DataplaneRegisteredName == name {
			rv = append(rv, clientConnection)
		}
	}

	return rv
}

func (m *ClientConnectionManager) GetClientConnectionBySource(networkServiceName string) []*model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()

	var rv []*model.ClientConnection
	for _, clientConnection := range clientConnections {
		if clientConnection.Request.IsRemote() {
			nsmConnection := clientConnection.Xcon.GetSource().(*crossconnect.CrossConnect_RemoteSource).RemoteSource
			if nsmConnection.SourceNetworkServiceManagerName == networkServiceName {
				rv = append(rv, clientConnection)
			}
		}
	}
	return rv
}

func (m *ClientConnectionManager) UpdateRemoteMonitorDone(networkServiceManagerName string) {
	// We need to be sure there is no active connections from selected Remote NSM.
	for _, conn := range m.GetClientConnectionBySource(networkServiceManagerName) {
		// Since remote monitor is done, and if connection is not closed we need to close them.
		m.manager.RemoteConnectionLost(conn)
	}
}

func (m *ClientConnectionManager) UpdateFromInitialState(xcons []*crossconnect.CrossConnect, dataplane *model.Dataplane) {
	m.manager.RestoreConnections(xcons, dataplane.RegisteredName)
}
