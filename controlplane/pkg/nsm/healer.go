package nsm

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
	"time"
)

type networkServiceHealer interface {
	healDstDown(healID string, connection *model.ClientConnection) bool
	healDataplaneDown(healID string, connection *model.ClientConnection) bool
	healDstUpdate(healID string, connection *model.ClientConnection) bool
	healDstNmgrDown(healID string, connection *model.ClientConnection) bool
}

type healer struct {
	serviceRegistry serviceregistry.ServiceRegistry
	model           model.Model

	nsm *networkServiceManager
}

func (h *healer) healDstDown(healID string, connection *model.ClientConnection) bool {
	logrus.Infof("NSM_Heal(1.1.1-%v) Checking if DST die is NSMD/DST die...", healID)
	// Check if this is a really HealState_DstDown or HealState_DstNmgrDown
	if !h.nsm.isLocalEndpoint(connection.Endpoint) {
		ctx, cancel := context.WithTimeout(context.Background(), h.nsm.GetHealProperties().HealTimeout*3)
		defer cancel()
		remoteNsmClient, err := h.nsm.createNSEClient(ctx, connection.Endpoint)
		if remoteNsmClient != nil {
			_ = remoteNsmClient.Cleanup()
		}
		if err != nil {
			// This is NSMD die case.
			logrus.Infof("NSM_Heal(1.1.2-%v) Connection healing state is %v...", healID, nsm.HealState_DstNmgrDown)
			return h.healDstNmgrDown(healID, connection)
		}
	}

	logrus.Infof("NSM_Heal(1.1.2-%v) Connection healing state is %v...", healID, nsm.HealState_DstDown)

	// Destination is down, we need to find it again.
	if connection.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		logrus.Infof("NSM_Heal(2.1-%v) Remote NSE heal is done on source side", healID)
		return false
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), h.nsm.GetHealProperties().HealTimeout*3)
		defer cancel()

		logrus.Infof("NSM_Heal(2.2-%v) Starting DST Heal...", healID)
		// We are client NSMd, we need to try recover our connection srv.

		// Wait for NSE not equal to down one, since we know it will be re-registered with new EndpointName.
		if !h.waitNSE(ctx, connection, connection.Endpoint.NetworkserviceEndpoint.EndpointName, connection.GetNetworkService()) {
			// Not remote NSE found, we need to update connection
			if dst := connection.Xcon.GetRemoteDestination(); dst != nil {
				dst.SetId("-") // We need to mark this as new connection.
			}
			if dst := connection.Xcon.GetLocalDestination(); dst != nil {
				dst.SetId("-") // We need to mark this as new connection.
			}
			// We need to remove selected endpoint here.
			connection.Endpoint = nil
		}
		// Fallback to heal with choose of new NSE.
		requestCtx, requestCancel := context.WithTimeout(context.Background(), h.nsm.GetHealProperties().HealRequestTimeout)
		defer requestCancel()
		logrus.Errorf("NSM_Heal(2.3.0-%v) Starting Heal by calling request: %v", healID, connection.Request)
		recoveredConnection, err := h.nsm.request(requestCtx, connection.Request, connection)
		if err != nil {
			logrus.Errorf("NSM_Heal(2.3.1-%v) Failed to heal connection: %v", healID, err)
			// We need to delete connection, since we are not able to Heal it
			h.model.DeleteClientConnection(connection.ConnectionId)
			if err != nil {
				logrus.Errorf("NSM_Heal(2.3.2-%v) Error in Recovery Close: %v", healID, err)
			}
		} else {
			logrus.Infof("NSM_Heal(2.4-%v) Heal: Connection recovered: %v", healID, recoveredConnection)
			return true
		}
	}
	// Let's Close remote connection and re-create new one.
	return false
}

func (h *healer) healDataplaneDown(healID string, connection *model.ClientConnection) bool {
	ctx, cancel := context.WithTimeout(context.Background(), h.nsm.GetHealProperties().HealTimeout)
	defer cancel()

	// Dataplane is down, we only need to re-programm dataplane.
	// 1. Wait for dataplane to appear.
	logrus.Infof("NSM_Heal(3.1-%v) Waiting for Dataplane to recovery...", healID)
	if err := h.serviceRegistry.WaitForDataplaneAvailable(h.model, h.nsm.GetHealProperties().HealDataplaneTimeout); err != nil {
		logrus.Errorf("NSM_Heal(3.1-%v) Dataplane is not available on recovery for timeout %v: %v", h.nsm.GetHealProperties().HealDataplaneTimeout, healID, err)
		return false
	}
	logrus.Infof("NSM_Heal(3.2-%v) Dataplane is now available...", healID)

	// We could send connection is down now.
	h.model.UpdateClientConnection(connection)

	if connection.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		// Recovery will be performed by NSM client side.
		logrus.Infof("NSM_Heal(3.3-%v)  Healing will be continued on source side...", healID)
		return true
	}

	// We have Dataplane now, let's try request all again.
	// Update request to contain a proper connection object from previous attempt.
	request := connection.Request.Clone()
	request.SetConnection(connection.GetConnectionSource())
	h.requestOrClose(fmt.Sprintf("NSM_Heal(3.4-%v) ", healID), ctx, request, connection)
	return true
}

func (h *healer) healDstUpdate(healID string, connection *model.ClientConnection) bool {
	ctx, cancel := context.WithTimeout(context.Background(), h.nsm.GetHealProperties().HealTimeout)
	defer cancel()

	// Remote DST is updated.
	// Update request to contain a proper connection object from previous attempt.
	logrus.Infof("NSM_Heal(5.1-%v) Healing Src Update... %v", healID, connection)
	if connection.Request != nil {
		request := connection.Request.Clone()
		request.SetConnection(connection.GetConnectionSource())

		h.requestOrClose(fmt.Sprintf("NSM_Heal(5.2-%v) ", healID), ctx, request, connection)
		return true
	}
	return false
}

func (h *healer) healDstNmgrDown(healID string, connection *model.ClientConnection) bool {
	logrus.Infof("NSM_Heal(6.1-%v) Starting DST + NSMGR Heal...", healID)

	ctx, cancel := context.WithTimeout(context.Background(), h.nsm.GetHealProperties().HealTimeout*3)
	defer cancel()

	var initialEndpoint = connection.Endpoint
	networkServiceName := connection.GetNetworkService()
	// Wait for exact same NSE to be available with NSMD connection alive.
	if connection.Endpoint != nil && !h.waitSpecificNSE(ctx, connection) {
		// Not remote NSE found, we need to update connection
		if dst := connection.Xcon.GetRemoteDestination(); dst != nil {
			dst.SetId("-") // We need to mark this as new connection.
		}
		if dst := connection.Xcon.GetLocalDestination(); dst != nil {
			dst.SetId("-") // We need to mark this as new connection.
		}
		connection.Endpoint = nil
	}
	requestCtx, requestCancel := context.WithTimeout(context.Background(), h.nsm.GetHealProperties().HealRequestTimeout)
	defer requestCancel()
	recoveredConnection, err := h.nsm.request(requestCtx, connection.Request, connection)
	if err != nil {
		logrus.Errorf("NSM_Heal(6.2.1-%v) Failed to heal connection with same NSE from registry: %v", healID, err)
		if initialEndpoint != nil {
			logrus.Infof("NSM_Heal(6.2.2-%v) Waiting for another NSEs...", healID)
			// In this case, most probable both NSMD and NSE are die, and registry was outdated on moment of waitNSE.
			if h.waitNSE(ctx, connection, initialEndpoint.NetworkserviceEndpoint.EndpointName, networkServiceName) {
				// Ok we have NSE, lets retry request
				requestCtx, requestCancel := context.WithTimeout(context.Background(), h.nsm.GetHealProperties().HealRequestTimeout)
				defer requestCancel()
				recoveredConnection, err = h.nsm.request(requestCtx, connection.Request, connection)
				if err != nil {
					if err != nil {
						logrus.Errorf("NSM_Heal(6.2.3-%v) Error in Recovery Close: %v", healID, err)
					}
				} else {
					logrus.Infof("NSM_Heal(6.3-%v) Heal: Connection recovered: %v", healID, recoveredConnection)
					return true
				}
			}
		}

		logrus.Errorf("NSM_Heal(6.4.1-%v) Failed to heal connection: %v", healID, err)
		// We need to delete connection, since we are not able to Heal it
		h.model.DeleteClientConnection(connection.ConnectionId)
		if err != nil {
			logrus.Errorf("NSM_Heal(6.4.2-%v) Error in Recovery Close: %v", healID, err)
		}
	} else {
		logrus.Infof("NSM_Heal(6.5-%v) Heal: Connection recovered: %v", healID, recoveredConnection)
		return true
	}
	return false
}

func (h *healer) requestOrClose(logPrefix string, ctx context.Context, request nsm.NSMRequest, clientConnection *model.ClientConnection) {
	logrus.Infof("%v delegate to Request %v", logPrefix, request)
	connection, err := h.nsm.request(ctx, request, clientConnection)
	if err != nil {
		logrus.Errorf("%v Failed to heal connection: %v", logPrefix, err)
		// Close in case of any errors in recovery.
		err = h.nsm.Close(context.Background(), clientConnection)
		logrus.Errorf("%v Error in Recovery Close: %v", logPrefix, err)
	} else {
		logrus.Infof("%v Heal: Connection recovered: %v", logPrefix, connection)
	}
}

func (h *healer) waitSpecificNSE(ctx context.Context, clientConnection *model.ClientConnection) bool {
	discoveryClient, err := h.serviceRegistry.DiscoveryClient()
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
					if h.nsm.checkUpdateNSE(ctx, clientConnection, ep, endpointResponse) {
						logrus.Infof("NSE is available and Remote NSMD is accessible. %s. Since elapsed: %v", clientConnection.Endpoint.NetworkServiceManager.Url, time.Since(st))
						// We are able to connect to NSM with required NSE
						return true
					}
				}
			}
		}

		if time.Since(st) > h.nsm.GetHealProperties().HealDSTNSEWaitTimeout {
			logrus.Errorf("Timeout waiting for NetworkService: %v and NSE: %v", networkService, clientConnection.Endpoint.NetworkserviceEndpoint.EndpointName)
			return false
		}
		// Wait a bit
		<-time.Tick(h.nsm.GetHealProperties().HealDSTNSEWaitTick)
	}
}

func (h *healer) waitNSE(ctx context.Context, clientConnection *model.ClientConnection, ignoreEndpoint string, networkService string) bool {
	discoveryClient, err := h.serviceRegistry.DiscoveryClient()
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
				if h.nsm.getNetworkServiceManagerName() == ep.GetNetworkServiceManagerName() {
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
				if h.nsm.checkUpdateNSE(ctx, clientConnection, ep, endpointResponse) {
					logrus.Infof("NSE is available and Remote NSMD is accessible. %s. Since elapsed: %v", clientConnection.Endpoint.NetworkServiceManager.Url, time.Since(st))
					// We are able to connect to NSM with required NSE
					return true
				}
			}
		}

		if time.Since(st) > h.nsm.GetHealProperties().HealDSTNSEWaitTimeout {
			logrus.Errorf("Timeout waiting for NetworkService: %v", networkService)
			return false
		}
		// Wait a bit
		<-time.Tick(h.nsm.GetHealProperties().HealDSTNSEWaitTick)
	}
}
