package services

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
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
func (m *ClientConnectionManager) UpdateXcon(cc nsm.ClientConnection, newXcon *crossconnect.CrossConnect) {
	if upd := m.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.Xcon = newXcon
	}); upd != nil {
		cc = upd
	} else {
		logrus.Errorf("Trying to update not existing connection: %v", cc.GetID())
		return
	}

	if src := newXcon.GetLocalSource(); src != nil && src.State == local.State_DOWN {
		logrus.Info("ClientConnection src state is down")
		ctx := context.Background()
		if modelCc := m.model.GetClientConnection(cc.GetID()); modelCc != nil {
			ctx = opentracing.ContextWithSpan(ctx, modelCc.Span)
		}
		err := m.manager.CloseConnection(ctx, cc)
		if err != nil {
			logrus.Error(err)
		}
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
func (m *ClientConnectionManager) LocalDestinationUpdated(cc *model.ClientConnection, localDst *local.Connection) {
	if cc.ConnectionState != model.ClientConnectionReady {
		return
	}

	// NSE is not aware of 'Workspace' and 'WorkspaceNSEName' connection mechanism parameters
	localDst.GetMechanism().GetParameters()[local.Workspace] =
		cc.Xcon.GetLocalDestination().GetMechanism().GetParameters()[local.Workspace]
	localDst.GetMechanism().GetParameters()[local.WorkspaceNSEName] =
		cc.Xcon.GetLocalDestination().GetMechanism().GetParameters()[local.WorkspaceNSEName]

	m.destinationUpdated(cc, localDst)
}

// RemoteDestinationUpdated handles case when remote connection parameters changed
func (m *ClientConnectionManager) RemoteDestinationUpdated(cc *model.ClientConnection, remoteDst *remote.Connection) {
	if cc.ConnectionState != model.ClientConnectionReady {
		return
	}

	if remoteDst.State == remote.State_UP {
		// TODO: in order to update connection parameters we need to update model here
		// We do not need to heal in case DST state is UP, remote NSM will try to recover and only when will send Update, Delete of connection.
		return
	}

	m.destinationUpdated(cc, remoteDst)
}

func (m *ClientConnectionManager) destinationUpdated(cc nsm.ClientConnection, dst connection.Connection) {
	// Check if it update we already have
	if dst.Equals(cc.GetConnectionDestination()) {
		// Since they are same, we do not need to do anything.
		return
	}

	if upd := m.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.Xcon.SetDestinationConnection(dst)
	}); upd != nil {
		cc = upd
	} else {
		logrus.Errorf("Trying to update not existing connection: %v", cc.GetID())
		return
	}

	m.manager.Heal(cc, nsm.HealStateDstUpdate)
}

// Since cross connect is ours, we could always use local connection id to identify client connection.
func (m *ClientConnectionManager) GetClientConnectionByXcon(xcon *crossconnect.CrossConnect) *model.ClientConnection {
	return m.model.GetClientConnection(xcon.GetSourceConnection().GetId())
}

// GetClientConnectionByLocalDst returns a ClientConnection with `Xcon.GetLocalDestination().GetID() == dstID`
// or `null` if there is no such connection
func (m *ClientConnectionManager) GetClientConnectionByLocalDst(dstID string) *model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()

	for _, clientConnection := range clientConnections {
		logrus.Infof("checking existing connection: %v to match %v", clientConnection.Xcon, dstID)
		if dst := clientConnection.Xcon.GetLocalDestination(); dst != nil && dst.GetId() == dstID {
			return clientConnection
		}
	}

	return nil
}

// GetClientConnectionByRemoteDst returns a ClientConnection with `Xcon.GetRemoteDestination().GetId() == dstID`
// or `null` if there is no such connection
func (m *ClientConnectionManager) GetClientConnectionByRemoteDst(dstID, remoteName string) *model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()

	for _, clientConnection := range clientConnections {
		logrus.Infof("checking existing connection: %v to match %v %v", clientConnection.Xcon, dstID, remoteName)
		if dst := clientConnection.Xcon.GetRemoteDestination(); dst != nil && dst.GetId() == dstID && dst.GetDestinationNetworkServiceManagerName() == remoteName {
			return clientConnection
		}
	}

	return nil
}

type channelConnectionListener struct {
	model.ListenerImpl
	channel chan *model.ClientConnection
}

// ClientConnectionUpdated will be called when ClientConnection in model is updated
func (c *channelConnectionListener) ClientConnectionUpdated(old, new *model.ClientConnection) {
	c.channel <- new
}

// ClientConnectionDeleted will be called when ClientConnection in model is deleted
func (c *channelConnectionListener) ClientConnectionDeleted(clientConnection *model.ClientConnection) {
	c.channel <- clientConnection
}

// WaitPendingConnections returns a ClientConnection with `Xcon.GetRemoteDestination().GetId() == dstID`
// or `null` if there is no such connection, before return it will wait
func (m *ClientConnectionManager) WaitPendingConnections(ctx context.Context, id, remoteName string) (*model.ClientConnection, error) {
	clientConnections := m.model.GetAllClientConnections()
	var pendingConnections []*model.ClientConnection
	for _, cc := range clientConnections {
		if m.isConnectionPending(cc) {
			pendingConnections = append(pendingConnections, cc)
		}
	}
	if len(pendingConnections) > 0 {
		//updateChannel := make(chan )
		listener := &channelConnectionListener{
			channel: make(chan *model.ClientConnection, 100),
		}
		m.model.AddListener(listener)
		defer m.model.RemoveListener(listener)

		for len(pendingConnections) > 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("Timeout during wait for connection with connectionId=%v and remoteName=%s", id, remoteName)
			case c := <-listener.channel:
				// If connection status is changed or it removed.
				if !m.isConnectionPending(c) || m.model.GetClientConnection(c.GetID()) == nil {
					// State changed, lets remove connection from list, but we not sure if this is ours.
					for idx, cc := range pendingConnections {
						if cc.ConnectionID == c.ConnectionID {
							pendingConnections = append(pendingConnections[:idx], pendingConnections[idx+1:]...)
							break
						}
					}
					if cc := m.GetClientConnectionByRemoteDst(id, remoteName); cc != nil {
						// We found our connection, lets' finish and return.
						return cc, nil
					}
				}
				break
			}
		}
		// in case it was pending, but it was deleted or not ours
		return nil, nil
	}
	// If no pending connections and we could not find, but lets check again.
	return nil, fmt.Errorf("No connection with id=%v and remoteName=%v are found", id, remoteName)
}

func (m *ClientConnectionManager) isConnectionPending(cc *model.ClientConnection) bool {
	return cc.ConnectionState == model.ClientConnectionRequesting || cc.ConnectionState == model.ClientConnectionHealing || cc.ConnectionState == model.ClientConnectionHealingBegin
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
		m.manager.RemoteConnectionLost(context.Background(), conn)
	}
}

func (m *ClientConnectionManager) UpdateFromInitialState(xcons []*crossconnect.CrossConnect, dataplane *model.Dataplane) {
	m.manager.RestoreConnections(xcons, dataplane.RegisteredName)
}
