package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

const (
	// deletedConnectionLifetime -  3 minutes to handle connections delete and update
	deletedConnectionLifetime = time.Minute * 3
)

type managedClientConnection struct {
	sync.RWMutex
	deleteTime       time.Time
	deleted          bool
	clientConnection *model.ClientConnection
}

type ClientConnectionManager struct {
	model           model.Model
	serviceRegistry serviceregistry.ServiceRegistry
	manager         nsm.NetworkServiceManager

	// A map of deleted connections.
	clientConnections sync.Map
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
	span := spanhelper.FromContext(ctx, "UpdateXcon")
	defer span.Finish()
	ctx = span.Context()
	span.LogObject("connection", cc)
	span.LogObject("newXCon", newXcon)
	logger := span.Logger()

	if upd := m.model.ApplyClientConnectionChanges(ctx, cc.GetID(), func(cc *model.ClientConnection) {
		//TODO: This should be passed to Healing and not updated here.
		cc.Xcon = newXcon
	}); upd != nil {
		cc = upd
	} else {
		err := errors.Errorf("trying to update not existing connection: %v", cc.GetID())
		span.LogError(err)
		return
	}

	if src := newXcon.GetLocalSource(); src != nil && src.State == local.State_DOWN {
		logger.Info("ClientConnection src state is down. Closing.")
		err := m.manager.CloseConnection(ctx, cc)
		span.LogError(err)
		return
	}

	if dst := newXcon.GetLocalDestination(); dst != nil && dst.State == local.State_DOWN {
		logger.Info("ClientConnection dst state is down. calling Heal.")
		m.manager.Heal(ctx, cc, nsm.HealStateDstDown)
		return
	}
}

// DestinationDown handles case when destination down
func (m *ClientConnectionManager) DestinationDown(ctx context.Context, cc nsm.ClientConnection, nsmdDie bool) {
	span := spanhelper.FromContext(ctx, "DestinationDown")
	defer span.Finish()
	ctx = span.Context()
	span.LogValue("nsmgr.die", nsmdDie)
	if nsmdDie {
		m.manager.Heal(ctx, cc, nsm.HealStateDstNmgrDown)
	} else {
		m.manager.Heal(ctx, cc, nsm.HealStateDstDown)
	}
}

// ForwarderDown handles case of local dp down
func (m *ClientConnectionManager) ForwarderDown(ctx context.Context, forwarder *model.Forwarder) {
	span := spanhelper.GetSpanHelper(ctx)
	ccs := m.model.GetAllClientConnections()
	for _, cc := range ccs {
		span.LogObject(fmt.Sprintf("ForwarderDeleted-%v", cc.GetID()), cc)
		if cc.ForwarderRegisteredName == forwarder.RegisteredName {
			span := common.SpanHelperFromConnection(ctx, cc, "ForwarderDown")
			defer span.Finish()
			span.LogObject("forwarder", forwarder)

			m.manager.Heal(span.Context(), cc, nsm.HealStateForwarderDown)
		}
	}
}

// LocalDestinationUpdated handles case when local connection parameters changed
func (m *ClientConnectionManager) LocalDestinationUpdated(ctx context.Context, cc *model.ClientConnection, localDst *local.Connection) {
	span := spanhelper.FromContext(ctx, "LocalDestinationUpdated")
	defer span.Finish()
	ctx = span.Context()
	span.LogObject("connection", cc)
	span.LogObject("destination", localDst)

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
	span := spanhelper.FromContext(ctx, "RemoteDestinationUpdated")
	defer span.Finish()
	ctx = span.Context()
	span.LogObject("connection", cc)
	span.LogObject("remoteDst", remoteDst)

	logger := span.Logger()
	if cc.ConnectionState != model.ClientConnectionReady {
		logger.Infof("Event not send... %v", cc.GetID())
		return
	}

	if remoteDst.State == remote.State_UP {
		logger.Infof("State is already UP do not send")
		// TODO: in order to update connection parameters we need to update model here
		// We do not need to heal in case DST state is UP, remote NSM will try to recover and only when will send Update, Delete of connection.
		return
	}

	logger.Infof("Event send... %v", cc.GetID())
	m.destinationUpdated(ctx, cc, remoteDst)
}

func (m *ClientConnectionManager) destinationUpdated(ctx context.Context, cc nsm.ClientConnection, dst connection.Connection) {
	span := spanhelper.FromContext(ctx, "destinationUpdated")
	defer span.Finish()
	ctx = span.Context()
	span.LogObject("connection", cc)
	span.LogObject("destination", dst)

	logger := span.Logger()
	// Check if it update we already have
	if dst.Equals(cc.GetConnectionDestination()) {
		logger.Infof("No event same destination objects")
		// Since they are same, we do not need to do anything.
		return
	}

	if upd := m.model.ApplyClientConnectionChanges(ctx, cc.GetID(), func(cc *model.ClientConnection) {
		cc.Xcon.SetDestinationConnection(dst)
	}); upd != nil {
		cc = upd
	} else {
		err := errors.Errorf("trying to update not existing connection: %v", cc.GetID())
		span.LogError(err)
		return
	}

	m.manager.Heal(ctx, cc, nsm.HealStateDstUpdate)
}

// GetClientConnectionByXcon - Since cross connect is ours, we could always use local connection id to identify client connection.
func (m *ClientConnectionManager) GetClientConnectionByXcon(xcon *crossconnect.CrossConnect) *model.ClientConnection {
	id := xcon.GetId()
	result := m.model.GetClientConnection(id)
	if result != nil {
		return result
	}
	if v, ok := m.clientConnections.Load(id); ok {
		if deleted, ok := v.(*managedClientConnection); ok {
			return deleted.clientConnection
		}
	}
	return nil
}

// GetClientConnectionByLocalDst returns a ClientConnection with `Xcon.GetLocalDestination().GetID() == dstID`
// or `null` if there is no such connection
func (m *ClientConnectionManager) GetClientConnectionByLocalDst(dstID string) *model.ClientConnection {
	clientConnections := m.getClientConnections()

	for _, clientConnection := range clientConnections {
		xCon := clientConnection.Xcon
		if xCon != nil {
			logrus.Infof("checking existing connection: %v to match %v", xCon, dstID)
			if dst := xCon.GetLocalDestination(); dst != nil && dst.GetId() == dstID {
				return clientConnection
			}
		}
	}

	return nil
}

// GetClientConnectionByRemoteDst returns a ClientConnection with `Xcon.GetRemoteDestination().GetId() == dstID`
// or `null` if there is no such connection
func (m *ClientConnectionManager) GetClientConnectionByRemoteDst(dstID, remoteName string) *model.ClientConnection {
	clientConnections := m.getClientConnections()
	for _, clientConnection := range clientConnections {
		xCon := clientConnection.Xcon
		if xCon != nil {
			logrus.Infof("checking existing connection: %v to match %v %v", xCon, dstID, remoteName)
			if dst := xCon.GetRemoteDestination(); dst != nil && dst.GetId() == dstID && dst.GetDestinationNetworkServiceManagerName() == remoteName {
				logrus.Infof("found remote connection %v", clientConnection)
				return clientConnection
			}
		}
	}

	logrus.Infof("NO DST found to match %v to match %v", dstID, remoteName)

	return nil
}

func (m *ClientConnectionManager) getClientConnections() []*model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()
	m.clientConnections.Range(func(k, v interface{}) bool {
		if cc, ok := v.(*managedClientConnection); ok {
			cc.Lock()
			clientConnections = append(clientConnections, cc.clientConnection)
			cc.Unlock()
		}
		return true
	})
	return clientConnections
}

type channelConnectionListener struct {
	model.ListenerImpl
	channel chan *model.ClientConnection
}

// ClientConnectionUpdated will be called when ClientConnection in model is updated
func (c *channelConnectionListener) ClientConnectionUpdated(ctx context.Context, old, new *model.ClientConnection) {
	c.channel <- new
}

// ClientConnectionDeleted will be called when ClientConnection in model is deleted
func (c *channelConnectionListener) ClientConnectionDeleted(ctx context.Context, clientConnection *model.ClientConnection) {
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
				return nil, errors.Errorf("timeout during wait for connection with connectionId=%v and remoteName=%s", id, remoteName)
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
			}
		}
		// in case it was pending, but it was deleted or not ours
		return nil, nil
	}
	// If no pending connections and we could not find, but lets check again.
	return nil, errors.Errorf("no connection with id=%v and remoteName=%v are found", id, remoteName)
}

func (m *ClientConnectionManager) isConnectionPending(cc *model.ClientConnection) bool {
	return cc.ConnectionState == model.ClientConnectionRequesting || cc.ConnectionState == model.ClientConnectionHealing || cc.ConnectionState == model.ClientConnectionHealingBegin
}

// GetClientConnectionByRemote - return client conndction by remote name
func (m *ClientConnectionManager) GetClientConnectionByRemote(nsm *registry.NetworkServiceManager) []*model.ClientConnection {
	clientConnections := m.getClientConnections()
	var result []*model.ClientConnection
	for _, clientConnection := range clientConnections {
		if clientConnection.RemoteNsm != nil && clientConnection.RemoteNsm.GetName() == nsm.GetName() {
			result = append(result, clientConnection)
		}
	}
	return result
}

func (m *ClientConnectionManager) GetClientConnectionsByForwarder(name string) []*model.ClientConnection {
	clientConnections := m.getClientConnections()

	var rv []*model.ClientConnection
	for _, clientConnection := range clientConnections {
		if clientConnection.ForwarderRegisteredName == name {
			rv = append(rv, clientConnection)
		}
	}

	return rv
}

// GetClientConnectionBySource - return client connection by source networkservice manager name
func (m *ClientConnectionManager) GetClientConnectionBySource(networkServiceManagerName string) []*model.ClientConnection {
	clientConnections := m.getClientConnections()

	var rv []*model.ClientConnection
	for _, clientConnection := range clientConnections {
		if clientConnection.Request != nil && clientConnection.Xcon != nil && clientConnection.Request.IsRemote() {
			nsmConnection := clientConnection.Xcon.GetRemoteSource()
			if nsmConnection != nil && nsmConnection.SourceNetworkServiceManagerName == networkServiceManagerName {
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

// UpdateFromInitialState - restore from forwarder init state request
func (m *ClientConnectionManager) UpdateFromInitialState(xcons []*crossconnect.CrossConnect, forwarder *model.Forwarder, manager nsm.MonitorManager) {
	m.manager.RestoreConnections(xcons, forwarder.RegisteredName, manager)
}

// MarkConnectionDeleted - put connection into map of deleted connections.
func (m *ClientConnectionManager) MarkConnectionDeleted(clientConnection *model.ClientConnection) {
	if v, ok := m.clientConnections.Load(clientConnection.GetID()); ok {
		if cc, ok := v.(*managedClientConnection); ok {
			cc.Lock()
			cc.deleteTime = time.Now()
			cc.deleted = true
			cc.clientConnection = clientConnection
			cc.Unlock()
		}
	}
	m.CleanupDeletedConnections()
}

// MarkConnectionUpdated - put connection into map of deleted connections.
func (m *ClientConnectionManager) MarkConnectionUpdated(clientConnection *model.ClientConnection) {
	if v, ok := m.clientConnections.Load(clientConnection.GetID()); ok {
		if cc, ok := v.(*managedClientConnection); ok {
			cc.Lock()
			cc.clientConnection = clientConnection
			cc.Unlock()
		}
	}
	m.CleanupDeletedConnections()
}

// CleanupDeletedConnections - cleanup deleted connections if timeout passed.
func (m *ClientConnectionManager) CleanupDeletedConnections() {
	// Iterate over connections to cleanup any orphaned.
	idSlice := []string{}
	m.clientConnections.Range(func(k, v interface{}) bool {
		if c, ok := v.(*managedClientConnection); ok {
			c.RLock()
			deleted := c.deleted
			c.RUnlock()
			if deleted && time.Since(c.deleteTime) > deletedConnectionLifetime {
				idSlice = append(idSlice, k.(string))
			}
		}
		return true
	})
	for _, id := range idSlice {
		m.clientConnections.Delete(id)
	}
}

// MarkConnectionAdded - remember we have connection to send events from.
func (m *ClientConnectionManager) MarkConnectionAdded(clientConnection *model.ClientConnection) {
	m.clientConnections.Store(clientConnection.GetID(), &managedClientConnection{clientConnection: clientConnection, deleted: false})
}
