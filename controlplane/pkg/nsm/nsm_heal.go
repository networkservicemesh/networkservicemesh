package nsm

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (srv *networkServiceManager) Heal(connection nsm.NSMClientConnection, healState nsm.HealState) {
	logrus.Infof("Heal %v", connection)

	ctx, cancel := context.WithTimeout(context.Background(), HealTimeout)
	defer cancel()
	clientConnection := connection.(*model.ClientConnection)
	if clientConnection.IsClosing {
		//means that we already invoke closing of remotes, nothing to do here
		return
	}

	switch healState {
	case nsm.HealState_DstDown:
		// Destination is down, we need to find it again.
		if clientConnection.Xcon.GetRemoteSource() != nil {
			// NSMd id remote one, we just need to close and return.
			break
		} else {
			// We are client NSMd, we need to try recover our connection.
			//srv.
			err := srv.close(ctx, clientConnection, false)
			if err != nil {
				logrus.Warnf("Ignored error during connection healing: %v", err)
			}
			recoveredConnection, err := srv.request(ctx, clientConnection.Request, clientConnection)
			logrus.Infof("Recovered: %v", recoveredConnection)
			if err != nil {
				logrus.Errorf("Failed to heal connection: %v", err)
				// We just need to close dataplane, since connection is already closed
				err = srv.closeDataplane(clientConnection)
				// We need to delete connection, since we are not able to Heal it
				srv.model.DeleteClientConnection(clientConnection.ConnectionId)
				if err != nil {
					logrus.Errorf("Error in Recovery Close: %v", err)
				}
			} else {
				logrus.Infof("Heal: Connection recovered: %v", connection)
			}
			return
		}

		// Let's Close remote connection and re-create new one.
	case nsm.HealState_DataplaneDown:
		// Dataplane is down, we only need to re-programm dataplane.
		// 1. Wait for dataplane to appear.
		if err := srv.serviceRegistry.WaitForDataplaneAvailable(srv.model, HealDataplaneTimeout); err != nil {
			logrus.Errorf("Dataplane is not available on recovery for timeout %v: %v", HealDataplaneTimeout, err)
			break
		}

		// We have dataplane now, let's try request all again.

		// Update request to contain a proper connection object from previous attempt.
		request := clientConnection.Request.Clone()
		request.SetConnection(clientConnection.GetSourceConnection())

		connection, err := srv.request(ctx, request, clientConnection)
		if err != nil {
			logrus.Errorf("Failed to heal connection: %v", err)
			// Close in any case
			err = srv.Close(context.Background(), clientConnection)
			logrus.Errorf("Error in Recovery Close: %v", err)
		} else {
			logrus.Infof("Heal: Connection recovered: %v", connection)
		}
		return
	}

	// Close both connection and dataplane
	err := srv.Close(context.Background(), clientConnection)
	if err != nil {
		logrus.Errorf("Error in Recovery: %v", err)
	}
}
