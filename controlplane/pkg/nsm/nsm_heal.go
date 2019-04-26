package nsm

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"time"
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
		healed = srv.healer.healDstDown(healID, clientConnection)
	case nsm.HealState_DataplaneDown:
		healed = srv.healer.healDataplaneDown(healID, clientConnection)
	case nsm.HealState_DstUpdate:
		healed = srv.healer.healDstUpdate(healID, clientConnection)
	case nsm.HealState_DstNmgrDown:
		healed = srv.healer.healDstNmgrDown(healID, clientConnection)
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
