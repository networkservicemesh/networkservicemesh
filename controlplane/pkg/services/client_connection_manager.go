package services

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	connection2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
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
	_= m.manager.Close(context.Background(), clientConnection)
}

func (m *ClientConnectionManager) UpdateClientConnectionDataplaneStateDown(clientConnections []*model.ClientConnection) {
	logrus.Info("ClientConnection src state is down because of Dataplane down.")
	for _, clientConnection := range clientConnections {
		clientConnection.Xcon.GetLocalSource().State = connection.State_DOWN
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
	} else if( clientConnection.Xcon.GetRemoteDestination() != nil) {
		clientConnection.Xcon.GetRemoteDestination().State = connection2.State_DOWN
	}
	m.model.UpdateClientConnection(clientConnection)
	m.manager.Heal(clientConnection, nsm.HealState_DstDown)
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
