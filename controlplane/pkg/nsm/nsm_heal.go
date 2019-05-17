package nsm

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

func (srv *networkServiceManager) Heal(connection nsm.NSMClientConnection, healState nsm.HealState) {
	healID := create_logid()
	logrus.Infof("NSM_Heal(1-%v) %v", healID, connection)

	cc := connection.(*model.ClientConnection)
	if cc.ConnectionState != model.ClientConnectionReady {
		//means that we already closing/healing
		return
	}

	if !srv.properties.HealEnabled {
		logrus.Infof("NSM_Heal Is Disabled/Closing connection %v", connection)

		err := srv.Close(context.Background(), cc)
		if err != nil {
			logrus.Errorf("NSM_Heal Error in Close: %v", err)
		}
		return
	}

	defer func() {
		logrus.Infof("NSM_Heal(1.1-%v) Connection %v healing state is finished...", healID, cc.GetID())
	}()

	srv.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.ConnectionState = model.ClientConnectionHealing
	})

	healed := false

	// 2 Choose heal style
	switch healState {
	case nsm.HealStateDstDown:
		healed = srv.healProcessor.healDstDown(healID, cc)
	case nsm.HealStateDataplaneDown:
		healed = srv.healProcessor.healDataplaneDown(healID, cc)
	case nsm.HealStateDstUpdate:
		healed = srv.healProcessor.healDstUpdate(healID, cc)
	case nsm.HealStateDstNmgrDown:
		healed = srv.healProcessor.healDstNmgrDown(healID, cc)
	}

	if healed {
		cc = srv.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
			cc.ConnectionState = model.ClientConnectionReady
		})
		return
	}

	// Close both connection and dataplane
	err := srv.Close(context.Background(), cc)
	if err != nil {
		logrus.Errorf("NSM_Heal(4-%v) Error in Recovery: %v", healID, err)
	}
}
