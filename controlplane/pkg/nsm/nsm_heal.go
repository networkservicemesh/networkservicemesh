package nsm

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (srv *networkServiceManager) Heal(connection nsm.NSMClientConnection, healState nsm.HealState) {
	healID := create_logid()
	logrus.Infof("NSM_Heal(1-%v) %v", healID, connection)

	clientConnection := connection.(*model.ClientConnection)
	if clientConnection.ConnectionState != model.ClientConnection_Ready {
		//means that we already closing/healing
		return
	}

	if !srv.properties.HealEnabled {
		logrus.Infof("NSM_Heal Is Disabled/Closing connection %v", connection)

		err := srv.Close(context.Background(), clientConnection)
		if err != nil {
			logrus.Errorf("NSM_Heal Error in Close: %v", err)
		}
		return
	}

	defer func() {
		logrus.Infof("NSM_Heal(1.1-%v) Connection %v healing state is finished...", healID, clientConnection.GetId())
		clientConnection.ConnectionState = model.ClientConnection_Ready
	}()

	clientConnection.ConnectionState = model.ClientConnection_Healing

	healed := false

	// 2 Choose heal style
	switch healState {
	case nsm.HealState_DstDown:
		healed = srv.healProcessor.healDstDown(healID, clientConnection)
	case nsm.HealState_DataplaneDown:
		healed = srv.healProcessor.healDataplaneDown(healID, clientConnection)
	case nsm.HealState_RemoteDataplaneDown:
		healed = srv.healProcessor.healRemoteDataplaneDown(healID, clientConnection)
	case nsm.HealState_DstNmgrDown:
		healed = srv.healProcessor.healDstNmgrDown(healID, clientConnection)
	}

	if healed {
		return
	}

	// Close both connection and dataplane
	err := srv.Close(context.Background(), clientConnection)
	if err != nil {
		logrus.Errorf("NSM_Heal(4-%v) Error in Recovery: %v", healID, err)
	}
}
