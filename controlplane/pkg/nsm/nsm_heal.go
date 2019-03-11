package nsm

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"time"
)

func (srv *networkServiceManager) Heal(connection nsm.NSMClientConnection, healState nsm.HealState) {
	healId := create_logid()
	logrus.Infof("NSM_Heal(1-%v) %v", healId, connection)

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
		logrus.Infof("NSM_Heal(1.1-%v) Connection %v healing state is finished...", healId, clientConnection.GetId())
		clientConnection.ConnectionState = model.ClientConnection_Ready
	}()

	clientConnection.ConnectionState = model.ClientConnection_Healing

	// 2 Choose heal style
	switch healState {
	case nsm.HealState_DstDown:
		// Destination is down, we need to find it again.
		if clientConnection.Xcon.GetRemoteSource() != nil {
			// NSMd id remote one, we just need to close and return.
			logrus.Infof("NSM_Heal(2.1-%v) Remote NSE heal is done on source side", healId)
			break
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), srv.properties.HealTimeout*3)
			defer cancel()

			logrus.Infof("NSM_Heal(2.2-%v) Starting DST Heal...", healId)
			// We are client NSMd, we need to try recover our connection srv.

			if srv.isLocalEndpoint(clientConnection.Endpoint) {
				// if NSE is DIE, on recovery it would be different NSE with different ID, so lets's just wait for any NSE with required name available
				// And filter same NSE as we had, since information about it could be outdated.
				srv.waitAnyNSE(clientConnection)
			} else {
				// Remote NSE let's wait for remote NSM providing NSE with our endpoint id is available via registry.
				// Get endpoints, do it every time since we do not know if list are changed or not.
				if !srv.waitRemoteNSE(ctx, clientConnection) {
					// Not remote NSE found, we need to update connection
					if dst := clientConnection.Xcon.GetRemoteDestination(); dst != nil {
						dst.SetId("-") // We need to mark this as new connection.
					}
					if dst := clientConnection.Xcon.GetLocalDestination(); dst != nil {
						dst.SetId("-") // We need to mark this as new connection.
					}
				}
			}
			// Fallback to heal with choose of new NSE.
			requestCtx, requestCancel := context.WithTimeout(context.Background(), srv.properties.HealRequestTimeout)
			defer requestCancel()
			recoveredConnection, err := srv.request(requestCtx, clientConnection.Request, clientConnection)
			if err != nil {
				logrus.Errorf("NSM_Heal(2.3.1-%v) Failed to heal connection: %v", healId, err)
				// We need to delete connection, since we are not able to Heal it
				srv.model.DeleteClientConnection(clientConnection.ConnectionId)
				if err != nil {
					logrus.Errorf("NSM_Heal(2.3.2-%v) Error in Recovery Close: %v", healId, err)
				}
				clientConnection.ConnectionState = model.ClientConnection_Closed

			} else {
				logrus.Infof("NSM_Heal(2.4-%v) Heal: Connection recovered: %v", healId, recoveredConnection)
				return
			}
		}

		// Let's Close remote connection and re-create new one.
	case nsm.HealState_DataplaneDown:
		ctx, cancel := context.WithTimeout(context.Background(), srv.properties.HealTimeout)
		defer cancel()

		// Dataplane is down, we only need to re-programm dataplane.
		// 1. Wait for dataplane to appear.
		logrus.Infof("NSM_Heal(3.1-%v) Waiting for Dataplane to recovery...", healId)
		if err := srv.serviceRegistry.WaitForDataplaneAvailable(srv.model, srv.properties.HealDataplaneTimeout); err != nil {
			logrus.Errorf("NSM_Heal(3.1-%v) Dataplane is not available on recovery for timeout %v: %v", srv.properties.HealDataplaneTimeout, healId, err)
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
		request.SetConnection(clientConnection.GetConnectionSource())
		srv.requestOrClose(fmt.Sprintf("NSM_Heal(3.4-%v) ", healId), ctx, request, clientConnection)
		return
	case nsm.HealState_DstUpdate:
		ctx, cancel := context.WithTimeout(context.Background(), srv.properties.HealTimeout)
		defer cancel()

		// Remote DST is updated.
		// Update request to contain a proper connection object from previous attempt.
		logrus.Infof("NSM_Heal(5.1-%v) Healing Src Update... %v", healId, clientConnection)
		if clientConnection.Request != nil {
			request := clientConnection.Request.Clone()
			request.SetConnection(clientConnection.GetConnectionSource())

			srv.requestOrClose(fmt.Sprintf("NSM_Heal(5.2-%v) ", healId), ctx, request, clientConnection)
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
	logrus.Infof("%v delegate to Request %v", logPrefix, request)
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
func (srv *networkServiceManager) waitAnyNSE(clientConnection *model.ClientConnection) {
	st := time.Now()
	for {
		logrus.Infof("Waiting for NSE with network service %s. Since elapsed: %v", clientConnection.Endpoint.NetworkService.Name, time.Since(st))
		ignored := map[string]*registry.NSERegistration{}
		ignored[clientConnection.Endpoint.NetworkserviceEndpoint.EndpointName] = clientConnection.Endpoint
		nsmConnection := srv.newConnection(clientConnection.Request)
		ep, err := srv.getEndpoint(context.Background(), nsmConnection, ignored)
		if err == nil && ep != nil {
			// We could call requires since we have some Endpoint to check with.
			break
		}
		if time.Since(st) > srv.properties.HealDSTNSEWaitTimeout {
			logrus.Errorf("")
			break
		}
		// Wait a bit
		<-time.Tick(srv.properties.HealDSTNSEWaitTick)
	}
}

func (srv *networkServiceManager) waitRemoteNSE(ctx context.Context, clientConnection *model.ClientConnection) bool {
	discoveryClient, err := srv.serviceRegistry.DiscoveryClient()
	if err != nil {
		logrus.Errorf("Failed to connect to Registry... %v", err)
		// Still try to recovery
		return false
	}

	st := time.Now()

	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: clientConnection.GetNetworkService(),
	}

	for {
		logrus.Infof("Waiting for NSE with network service %s. Since elapsed: %v", clientConnection.Endpoint.NetworkService.Name, time.Since(st))

		endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
		if err == nil {
			for _, ep := range endpointResponse.NetworkServiceEndpoints {
				if ep.EndpointName == clientConnection.Endpoint.NetworkserviceEndpoint.EndpointName {
					// Out endpoint, we need to check if it is remote one and NSM is accessible.
					pingCtx, pingCancel := context.WithTimeout(ctx, srv.properties.HealRequestTimeout)
					defer pingCancel()
					reg := &registry.NSERegistration{
						NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[ep.GetNetworkServiceManagerName()],
						NetworkserviceEndpoint: ep,
						NetworkService:         endpointResponse.GetNetworkService(),
					}

					client, err := srv.createNSEClient(pingCtx, reg)
					if err == nil && client != nil {
						_ = client.Cleanup()
						// We are able to connect to NSM with required NSE
						return true
					}
				}
			}
		}

		if time.Since(st) > srv.properties.HealDSTNSEWaitTimeout {
			logrus.Errorf("Timeout waiting for NSE: %v", clientConnection.Endpoint.NetworkserviceEndpoint.EndpointName)
			return false
		}
		// Wait a bit
		<-time.Tick(srv.properties.HealDSTNSEWaitTick)
	}
}
