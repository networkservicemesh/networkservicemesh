package services

import (
	"context"
	"github.com/gogo/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
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

func (m *ClientConnectionManager) UpdateClientConnection(clientConnection *model.ClientConnection) {
	m.model.UpdateClientConnection(clientConnection)
}

func (m *ClientConnectionManager) UpdateClientConnectionSrcStateDown(clientConnection *model.ClientConnection) {
	logrus.Info("ClientConnection src state is down")
	clientConnection.Xcon.GetLocalSource().State = connection.State_DOWN
	m.model.UpdateClientConnection(clientConnection)
	_ = m.manager.Close(context.Background(), clientConnection)
}

func (m *ClientConnectionManager) UpdateClientConnectionDataplaneStateDown(clientConnections []*model.ClientConnection) {
	logrus.Info("ClientConnection src state is down because of Dataplane down.")
	for _, clientConnection := range clientConnections {
		m.markSourceConnectionDown(clientConnection)
		m.model.UpdateClientConnection(clientConnection)
	}
	for _, clientConnection := range clientConnections {
		m.manager.Heal(clientConnection, nsm.HealState_DataplaneDown)
	}
}

func (m *ClientConnectionManager) UpdateClientConnectionDstStateDown(clientConnection *model.ClientConnection) {
	logrus.Info("ClientConnection dst state is down")
	if clientConnection.Xcon.GetLocalDestination() != nil {
		clientConnection.Xcon.GetLocalDestination().State = connection.State_DOWN
	} else if clientConnection.Xcon.GetRemoteDestination() != nil {
		clientConnection.Xcon.GetRemoteDestination().State = remote_connection.State_DOWN
	}
	m.model.UpdateClientConnection(clientConnection)
	m.manager.Heal(clientConnection, nsm.HealState_DstDown)
}
func (m *ClientConnectionManager) UpdateClientConnectionDstUpdated(clientConnection *model.ClientConnection, remoteConnection *remote_connection.Connection) {
	if clientConnection.ConnectionState == model.ClientConnection_Healing || clientConnection.ConnectionState == model.ClientConnection_Requesting {
		// Do not need to process but we need to put update event is recieved.
		clientConnection.UpdateRecieved = true
		return
	}
	if clientConnection.ConnectionState != model.ClientConnection_Ready {
		return
	}

	// Check if it update we already have
	if proto.Equal(remoteConnection, clientConnection.Xcon.GetRemoteDestination()) {
		// Since they are same, we do not need to do anything.
		return
	}
	/*
		Lets treat source as down, since we are do know if connection is alive at this moment.
	 	Example Remote Dataplane die, could lead to new Remote dataplane with different Ip, so Remote connection is in restore state.
	*/
	m.markSourceConnectionDown(clientConnection)
	clientConnection.Xcon.Destination = &crossconnect.CrossConnect_RemoteDestination{
		RemoteDestination: remoteConnection,
	}
	m.model.UpdateClientConnection(clientConnection)
	m.manager.Heal(clientConnection, nsm.HealState_DstUpdate)
}

func (m *ClientConnectionManager) markSourceConnectionDown(clientConnection *model.ClientConnection) {
	if clientConnection.Xcon.GetRemoteSource() != nil {
		clientConnection.Xcon.GetRemoteSource().State = remote_connection.State_DOWN
	} else if clientConnection.Xcon.GetLocalSource() != nil {
		clientConnection.Xcon.GetLocalSource().State = connection.State_DOWN
	}
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

func (m *ClientConnectionManager) GetClientConnectionsByDataplane(name string) []*model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()

	var rv []*model.ClientConnection
	for _, clientConnection := range clientConnections {
		if clientConnection.Dataplane.RegisteredName == name {
			rv = append(rv, clientConnection)
		}
	}

	return rv
}

func (m *ClientConnectionManager) DeleteClientConnection(clientConnection *model.ClientConnection) {
	m.model.DeleteClientConnection(clientConnection.ConnectionId)
}
