package nsm

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (srv *networkServiceManager) Heal(connection nsm.NSMClientConnection, healState nsm.HealState) {
	healId := create_logid()
	logrus.Infof("NSM_Heal(1-%v) %v", healId, connection)

	clientConnection := connection.(*model.ClientConnection)
	if clientConnection.ConnectionState != model.ClientConnection_Ready {
		//means that we already closing/healing
		return
	}

	defer func(){
		logrus.Infof("NSM_Heal(1.1-%v) Connection %v healing state is finished...", healId, clientConnection.GetId())
		clientConnection.ConnectionState = model.ClientConnection_Ready
	}()

	clientConnection.ConnectionState = model.ClientConnection_Healing

	ctx, cancel := context.WithTimeout(context.Background(), HealTimeout)
	defer cancel()

	// 2 Choose heal style
	switch healState {
	case nsm.HealState_DstDown:
		// Destination is down, we need to find it again.
		if clientConnection.Xcon.GetRemoteSource() != nil {
			// NSMd id remote one, we just need to close and return.
			logrus.Infof("NSM_Heal(2.1-%v) Remote NSE heal is done on source side", healId)
			break
		} else {
			// We are client NSMd, we need to try recover our connection.
			//srv.
			err := srv.close(ctx, clientConnection, false)
			if err != nil {
				logrus.Warnf("NSM_Heal(2.2-%v) Ignored error during connection healing: %v", healId, err)
			}
			recoveredConnection, err := srv.request(ctx, clientConnection.Request, clientConnection)
			logrus.Infof("NSM_Heal(2.3-%v) Recovered: %v", healId, recoveredConnection)
			if err != nil {
				logrus.Errorf("NSM_Heal(2.3.1-%v) Failed to heal connection: %v", healId, err)
				// We just need to close dataplane, since connection is already closed
				err = srv.closeDataplane(clientConnection)
				// We need to delete connection, since we are not able to Heal it
				srv.model.DeleteClientConnection(clientConnection.ConnectionId)
				if err != nil {
					logrus.Errorf("NSM_Heal(2.3.2-%v) Error in Recovery Close: %v", healId, err)
				}
				clientConnection.ConnectionState = model.ClientConnection_Closed
			} else {
				logrus.Infof("NSM_Heal(2.4-%v) Heal: Connection recovered: %v", healId, connection)
			}
			return
		}

		// Let's Close remote connection and re-create new one.
	case nsm.HealState_DataplaneDown:
		// Dataplane is down, we only need to re-programm dataplane.
		// 1. Wait for dataplane to appear.
		logrus.Infof("NSM_Heal(3.1-%v) Waiting for Dataplane to recovery...", healId)
		if err := srv.serviceRegistry.WaitForDataplaneAvailable(srv.model, HealDataplaneTimeout); err != nil {
			logrus.Errorf("NSM_Heal(3.1-%v) Dataplane is not available on recovery for timeout %v: %v", HealDataplaneTimeout, healId, err)
			break
		}
		logrus.Infof("NSM_Heal(3.2-%v) Dataplane is now available...", healId)

		// We could send connection is down now.
		srv.model.UpdateClientConnection(clientConnection)

		if clientConnection.Xcon.GetRemoteSource() != nil {
			// NSMd id remote one, we just need to close and return.
			// Recovery will be performed by NSM client side.
			logrus.Infof("NSM_Heal(3.3-%v)  Healing will be continued on source side...", healId)
			return
		}

		// We have Dataplane now, let's try request all again.
		// Update request to contain a proper connection object from previous attempt.
		request := clientConnection.Request.Clone()
		request.SetConnection(clientConnection.GetSourceConnection())
		srv.requestOrClose(fmt.Sprintf("NSM_Heal(3.4-%v) ", healId) ,ctx, request, clientConnection)
		return
	case nsm.HealState_DstUpdate:
		// Remote DST is updated.
		// Update request to contain a proper connection object from previous attempt.
		logrus.Infof("NSM_Heal(4.1-%v) Healing DST Update/Remote Dataplane... %v", healId, clientConnection)
		if clientConnection.Request != nil {
			request := clientConnection.Request.Clone()
			request.SetConnection(clientConnection.GetSourceConnection())

			srv.requestOrClose(fmt.Sprintf("NSM_Heal(4.2-%v) ", healId), ctx, request, clientConnection)
			return
		}
	}

	// Close both connection and dataplane
	err := srv.Close(context.Background(), clientConnection)
	if err != nil {
		logrus.Errorf("NSM_Heal(4-%v) Error in Recovery: %v", healId, err)
	}

}

func (srv *networkServiceManager) requestOrClose(logPrefix string, ctx context.Context, request nsm.NSMRequest, clientConnection *model.ClientConnection) {
	logrus.Infof("%v delegate to Request %v", logPrefix, request )
	connection, err := srv.request(ctx, request, clientConnection)
	if err != nil {
		logrus.Errorf("%v Failed to heal connection: %v", logPrefix, err)
		// Close in case of any errors in recovery.
		err = srv.Close(context.Background(), clientConnection)
		logrus.Errorf("%v Error in Recovery Close: %v", logPrefix, err)
	} else {
		logrus.Infof("%v Heal: Connection recovered: %v", logPrefix, connection)
	}
}
