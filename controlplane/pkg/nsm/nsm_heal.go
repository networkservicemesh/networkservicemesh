package nsm

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (srv *networkServiceManager) Heal(clientConnection *model.ClientConnection, healState nsm.HealState) {
	logrus.Infof("Heal %v", clientConnection)

	if clientConnection.IsClosing {
		//means that we already invoke closing of remotes, nothing to do here
		return
	}
	clientConnection.IsClosing = true

	switch healState {
	case nsm.HealState_DstDown:
		// Destination is down.
		// Let's Close remote connection and re-create new one.
	case nsm.HealState_DataplaneDown:
		// Source is down, lets check if this is dataplane down and we could heal.
	}

	ls := clientConnection.Xcon.GetLocalSource()
	var err error
	if ls != nil {
		err = srv.Close(context.Background(), ls)
	} else {
		err = srv.Close(context.Background(), clientConnection.Xcon.GetRemoteSource())
	}
	if err != nil {
		logrus.Errorf("Error in Recovery: %v", err)
	}
}
