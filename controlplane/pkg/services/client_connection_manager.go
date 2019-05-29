package services

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
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
func (m *ClientConnectionManager) UpdateXcon(cc nsm.NSMClientConnection, newXcon *crossconnect.CrossConnect) {
	m.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.Xcon = newXcon
	})

	if src := newXcon.GetLocalSource(); src != nil && src.State == local_connection.State_DOWN {
		logrus.Info("ClientConnection src state is down")
		_ = m.manager.Close(context.Background(), cc)
		return
	}

	if dst := newXcon.GetLocalDestination(); dst != nil && dst.State == local_connection.State_DOWN {
		logrus.Info("ClientConnection dst state is down")
		m.manager.Heal(cc, nsm.HealState_DstDown)
		return
	}
}

// RemoteDestinationDown handles case when remote destination down
func (m *ClientConnectionManager) RemoteDestinationDown(cc nsm.NSMClientConnection, nsmdDie bool) {
	if nsmdDie {
		m.manager.Heal(cc, nsm.HealState_DstNmgrDown)
	} else {
		m.manager.Heal(cc, nsm.HealState_DstDown)
	}
}

// DataplaneDown handles case of local dp down
func (m *ClientConnectionManager) DataplaneDown(dataplane *model.Dataplane) {
	ccs := m.model.GetAllClientConnections()

	for _, cc := range ccs {
		if cc.DataplaneRegisteredName == dataplane.RegisteredName {
			m.manager.Heal(cc, nsm.HealState_DataplaneDown)
		}
	}
}

// RemoteDestinationUpdated handles case when remote connection parameters changed
func (m *ClientConnectionManager) RemoteDestinationUpdated(cc *model.ClientConnection, remoteConnection *remote_connection.Connection) {
	if cc.ConnectionState != model.ClientConnectionReady {
		return
	}

	if remoteConnection.State == remote_connection.State_UP {
		// TODO: in order to update connection parameters we need to update model here
		// We do not need to heal in case DST state is UP, remote NSM will try to recover and only when will send Update, Delete of connection.
		return
	}

	// Check if it update we already have
	if proto.Equal(remoteConnection, cc.Xcon.GetRemoteDestination()) {
		// Since they are same, we do not need to do anything.
		return
	}

	m.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.Xcon.Destination = &crossconnect.CrossConnect_RemoteDestination{
			RemoteDestination: remoteConnection,
		}
	})
	m.manager.Heal(cc, nsm.HealState_RemoteDataplaneDown)
}

func (m *ClientConnectionManager) GetClientConnectionByXcon(xcon *crossconnect.CrossConnect) *model.ClientConnection {
	var connectionId string
	if conn := xcon.GetLocalSource(); conn != nil {
		connectionId = conn.GetId()
	} else {
		connectionId = xcon.GetRemoteSource().GetId()
	}
	return m.model.GetClientConnection(connectionId)
}

func (m *ClientConnectionManager) GetClientConnectionByDst(dstId string) *model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()

	for _, clientConnection := range clientConnections {
		var destinationId string
		if dst := clientConnection.Xcon.GetLocalDestination(); dst != nil {
			destinationId = dst.GetId()
		} else {
			destinationId = clientConnection.Xcon.GetRemoteDestination().GetId()
		}

		if destinationId == dstId {
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
