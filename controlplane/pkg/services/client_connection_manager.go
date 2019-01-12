package services

import (
	"context"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
)

type ClientConnectionManager struct {
	model           model.Model
	serviceRegistry serviceregistry.ServiceRegistry
}

func NewClientConnectionManager(model model.Model, serviceRegistry serviceregistry.ServiceRegistry) *ClientConnectionManager {
	return &ClientConnectionManager{
		model:           model,
		serviceRegistry: serviceRegistry,
	}
}

func (m *ClientConnectionManager) UpdateClientConnectionState(clientConnection *model.ClientConnection, state crossconnect.CrossConnectState) {
	clientConnection.Xcon.State = state
	m.model.UpdateClientConnection(clientConnection)

	switch state {
	case crossconnect.CrossConnectState_SRC_DOWN:
		m.CloseRemotes(clientConnection, true, true)
	case crossconnect.CrossConnectState_DST_DOWN:
		m.CloseRemotes(clientConnection, true, false)
	}
}

func (m *ClientConnectionManager) CloseRemotes(clientConnection *model.ClientConnection,
	closeDataplane bool, closeEndpoint bool) {
	if clientConnection.IsClosing {
		//means that we already invoke closing of remotes, nothing to do here
		return
	}
	clientConnection.IsClosing = true
	if closeEndpoint {
		err := m.CloseXconOnEndpoint(clientConnection)
		if err != nil {
			logrus.Error(err)
		}
	}
	if closeDataplane {
		err := m.CloseXconOnDataplane(clientConnection)
		if err != nil {
			logrus.Error()
		}
	}
}

func (m *ClientConnectionManager) DeleteClientConnection(clientConnection *model.ClientConnection,
	closeDataplane bool, closeEndpoint bool) {
	logrus.Info("Deleting client connection...")
	m.CloseRemotes(clientConnection, closeDataplane, closeEndpoint)
	m.model.DeleteClientConnection(clientConnection.ConnectionId)
}

func (m *ClientConnectionManager) GetClientConnectionByXcon(xconId string) *model.ClientConnection {
	clientConnections := m.model.GetAllClientConnections()

	for _, clientConnection := range clientConnections {
		if clientConnection.Xcon.Id == xconId {
			return clientConnection
		}
	}

	return nil
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

func (m *ClientConnectionManager) CloseXconOnDataplane(clientConnection *model.ClientConnection) error {
	logrus.Info("Closing cross connection on dataplane...")
	dataplaneClient, conn, err := m.serviceRegistry.DataplaneConnection(clientConnection.Dataplane)
	if err != nil {
		logrus.Error(err)
		return err
	}
	if conn != nil {
		defer conn.Close()
	}
	if _, err := dataplaneClient.Close(context.Background(), clientConnection.Xcon); err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Info("Cross connection successfully closed on dataplane")
	return nil
}

func (m *ClientConnectionManager) CloseXconOnEndpoint(clientConnection *model.ClientConnection) error {
	if clientConnection.RemoteNsm != nil {
		remoteClient, conn, err := m.serviceRegistry.RemoteNetworkServiceClient(clientConnection.RemoteNsm)
		if err != nil {
			logrus.Error(err)
			return err
		}
		if conn != nil {
			defer conn.Close()
		}
		logrus.Info("Remote client successfully created")

		if _, err := remoteClient.Close(context.Background(), clientConnection.Xcon.GetRemoteDestination()); err != nil {
			logrus.Error(err)
			return err
		}
		logrus.Info("Remote part of cross connection successfully closed")
	} else {
		endpointClient, conn, err := m.serviceRegistry.EndpointConnection(clientConnection.Endpoint)
		if err != nil {
			logrus.Error(err)
			return err
		}
		if conn != nil {
			defer conn.Close()
		}

		logrus.Info("Closing NSE connection...")
		if _, err := endpointClient.Close(context.Background(), clientConnection.Xcon.GetLocalDestination()); err != nil {
			logrus.Error(err)
			return err
		}
		logrus.Info("NSE connection successfully closed")
	}

	return nil
}
