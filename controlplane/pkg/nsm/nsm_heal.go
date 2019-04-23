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

	// We need to check if Remove NSM is alive, and if not alive we should have a bit different wait logic here.
	if healState == nsm.HealState_DstDown {
		logrus.Infof("NSM_Heal(1.1.1-%v) Checking if DST die is NSMD/DST die...", healId)
		// Check if this is a really HealState_DstDown or HealState_DstNmgrDown
		if !srv.isLocalEndpoint(clientConnection.Endpoint) {
			ctx, cancel := context.WithTimeout(context.Background(), srv.properties.HealTimeout*3)
			defer cancel()
			remoteNsmClient, err := srv.createNSEClient(ctx, clientConnection.Endpoint)
			if remoteNsmClient != nil {
				_ = remoteNsmClient.Cleanup()
			}
			if err != nil {
				// This is NSMD die case.
				healState = nsm.HealState_DstNmgrDown
			}
		}
		logrus.Infof("NSM_Heal(1.1.2-%v) Connection healing state is %v...", healId, healState)
	}

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

			// Wait for NSE not equal to down one, since we know it will be re-registered with new EndpointName.
			if !srv.waitNSE(ctx, clientConnection, clientConnection.Endpoint.NetworkserviceEndpoint.EndpointName, clientConnection.GetNetworkService()) {
				// Not remote NSE found, we need to update connection
				if dst := clientConnection.Xcon.GetRemoteDestination(); dst != nil {
					dst.SetId("-") // We need to mark this as new connection.
				}
				if dst := clientConnection.Xcon.GetLocalDestination(); dst != nil {
					dst.SetId("-") // We need to mark this as new connection.
				}
				// We need to remove selected endpoint here.
				clientConnection.Endpoint = nil
			}
			// Fallback to heal with choose of new NSE.
			requestCtx, requestCancel := context.WithTimeout(context.Background(), srv.properties.HealRequestTimeout)
			defer requestCancel()
			logrus.Errorf("NSM_Heal(2.3.0-%v) Starting Heal by calling request: %v", healId, clientConnection.Request)
			recoveredConnection, err := srv.request(requestCtx, clientConnection.Request, clientConnection)
			if err != nil {
				logrus.Errorf("NSM_Heal(2.3.1-%v) Failed to heal connection: %v", healId, err)
				// We need to delete connection, since we are not able to Heal it
				srv.model.DeleteClientConnection(clientConnection.ConnectionId)
				if err != nil {
					logrus.Errorf("NSM_Heal(2.3.2-%v) Error in Recovery Close: %v", healId, err)
				}
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
	case nsm.HealState_DstNmgrDown:
		logrus.Infof("NSM_Heal(6.1-%v) Starting DST + NSMGR Heal...", healId)

		ctx, cancel := context.WithTimeout(context.Background(), srv.properties.HealTimeout*3)
		defer cancel()

		var initialEndpoint = clientConnection.Endpoint
		networkServiceName := clientConnection.GetNetworkService()
		// Wait for exact same NSE to be available with NSMD connection alive.
		if clientConnection.Endpoint != nil && !srv.waitSpecificNSE(ctx, clientConnection) {
			// Not remote NSE found, we need to update connection
			if dst := clientConnection.Xcon.GetRemoteDestination(); dst != nil {
				dst.SetId("-") // We need to mark this as new connection.
			}
			if dst := clientConnection.Xcon.GetLocalDestination(); dst != nil {
				dst.SetId("-") // We need to mark this as new connection.
			}
			clientConnection.Endpoint = nil
		}
		requestCtx, requestCancel := context.WithTimeout(context.Background(), srv.properties.HealRequestTimeout)
		defer requestCancel()
		recoveredConnection, err := srv.request(requestCtx, clientConnection.Request, clientConnection)
		if err != nil {
			logrus.Errorf("NSM_Heal(6.2.1-%v) Failed to heal connection with same NSE from registry: %v", healId, err)
			if initialEndpoint != nil {
				logrus.Infof("NSM_Heal(6.2.2-%v) Waiting for another NSEs...", healId)
				// In this case, most probable both NSMD and NSE are die, and registry was outdated on moment of waitNSE.
				if srv.waitNSE(ctx, clientConnection, initialEndpoint.NetworkserviceEndpoint.EndpointName, networkServiceName) {
					// Ok we have NSE, lets retry request
					requestCtx, requestCancel := context.WithTimeout(context.Background(), srv.properties.HealRequestTimeout)
					defer requestCancel()
					recoveredConnection, err = srv.request(requestCtx, clientConnection.Request, clientConnection)
					if err != nil {
						if err != nil {
							logrus.Errorf("NSM_Heal(6.2.3-%v) Error in Recovery Close: %v", healId, err)
						}
					} else {
						logrus.Infof("NSM_Heal(6.3-%v) Heal: Connection recovered: %v", healId, recoveredConnection)
						return
					}
				}
			}

			logrus.Errorf("NSM_Heal(6.4.1-%v) Failed to heal connection: %v", healId, err)
			// We need to delete connection, since we are not able to Heal it
			srv.model.DeleteClientConnection(clientConnection.ConnectionId)
			if err != nil {
				logrus.Errorf("NSM_Heal(6.4.2-%v) Error in Recovery Close: %v", healId, err)
			}
		} else {
			logrus.Infof("NSM_Heal(6.5-%v) Heal: Connection recovered: %v", healId, recoveredConnection)
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


func (srv *networkServiceManager) waitSpecificNSE(ctx context.Context, clientConnection *model.ClientConnection) bool {
	discoveryClient, err := srv.serviceRegistry.DiscoveryClient()
	if err != nil {
		logrus.Errorf("Failed to connect to Registry... %v", err)
		// Still try to recovery
		return false
	}

	st := time.Now()

	networkService := clientConnection.Endpoint.NetworkService.Name
	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: networkService,
	}

	defer func() {
		logrus.Infof("Complete Waiting for Remote NSE/NSMD with network service %s. Since elapsed: %v", networkService, time.Since(st))
	}()

	for {
		logrus.Infof("NSM: RemoteNSE: Waiting for NSE with network service %s NSE %v. Since elapsed: %v", networkService, clientConnection.Endpoint.NetworkserviceEndpoint.EndpointName, time.Since(st))

		endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
		if err == nil {
			for _, ep := range endpointResponse.NetworkServiceEndpoints {
				if ep.EndpointName == clientConnection.Endpoint.NetworkserviceEndpoint.EndpointName {
					// Out endpoint, we need to check if it is remote one and NSM is accessible.
					// Check remote is accessible.
					if srv.checkUpdateNSE(ctx, clientConnection, ep, endpointResponse) {
						logrus.Infof("NSE is available and Remote NSMD is accessible. %s. Since elapsed: %v", clientConnection.Endpoint.NetworkServiceManager.Url, time.Since(st))
						// We are able to connect to NSM with required NSE
						return true
					}
				}
			}
		}

		if time.Since(st) > srv.properties.HealDSTNSEWaitTimeout {
			logrus.Errorf("Timeout waiting for NetworkService: %v and NSE: %v", networkService, clientConnection.Endpoint.NetworkserviceEndpoint.EndpointName)
			return false
		}
		// Wait a bit
		<-time.Tick(srv.properties.HealDSTNSEWaitTick)
	}
}

func (srv *networkServiceManager) waitNSE(ctx context.Context, clientConnection *model.ClientConnection, ignoreEndpoint string, networkService string) bool {
	discoveryClient, err := srv.serviceRegistry.DiscoveryClient()
	if err != nil {
		logrus.Errorf("Failed to connect to Registry... %v", err)
		// Still try to recovery
		return false
	}

	st := time.Now()

	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: networkService,
	}

	defer func() {
		logrus.Infof("Complete Waiting for Remote NSE/NSMD with network service %s. Since elapsed: %v", networkService, time.Since(st))
	}()

	for {
		logrus.Infof("NSM: RemoteNSE: Waiting for NSE with network service %s. Since elapsed: %v", networkService, time.Since(st))

		endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
		if err == nil {
			for _, ep := range endpointResponse.NetworkServiceEndpoints {
				if ignoreEndpoint != "" && ep.EndpointName == ignoreEndpoint {
					// Skip ignored endpoint
					continue
				}

				// Check local only if not waiting for specific NSE.
				if srv.getNetworkServiceManagerName() == ep.GetNetworkServiceManagerName() {
					// Another local endpoint is found, success.
					reg := &registry.NSERegistration{
						NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[ep.GetNetworkServiceManagerName()],
						NetworkserviceEndpoint: ep,
						NetworkService:         endpointResponse.GetNetworkService(),
					}
					clientConnection.Endpoint = reg
					return true
				}
				// Check remote is accessible.
				if srv.checkUpdateNSE(ctx, clientConnection, ep, endpointResponse) {
					logrus.Infof("NSE is available and Remote NSMD is accessible. %s. Since elapsed: %v", clientConnection.Endpoint.NetworkServiceManager.Url, time.Since(st))
					// We are able to connect to NSM with required NSE
					return true
				}
			}
		}

		if time.Since(st) > srv.properties.HealDSTNSEWaitTimeout {
			logrus.Errorf("Timeout waiting for NetworkService: %v", networkService)
			return false
		}
		// Wait a bit
		<-time.Tick(srv.properties.HealDSTNSEWaitTick)
	}
}
