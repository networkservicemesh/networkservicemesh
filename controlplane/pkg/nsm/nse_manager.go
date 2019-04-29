package nsm

import (
	"context"
	"fmt"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
)

type networkServiceEndpointManager interface {
	getEndpoint(ctx context.Context, requestConnection nsm.NSMConnection, ignore_endpoints map[string]*registry.NSERegistration) (*registry.NSERegistration, error)
	createNSEClient(ctx context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceClient, error)
	isLocalEndpoint(endpoint *registry.NSERegistration) bool
	checkUpdateNSE(ctx context.Context, clientConnection *model.ClientConnection, ep *registry.NetworkServiceEndpoint, endpointResponse *registry.FindNetworkServiceResponse) bool
}

type nseManager struct {
	serviceRegistry serviceregistry.ServiceRegistry
	model           model.Model
	properties      *nsm.NsmProperties
}

func (nsem *nseManager) getEndpoint(ctx context.Context, requestConnection nsm.NSMConnection, ignore_endpoints map[string]*registry.NSERegistration) (*registry.NSERegistration, error) {

	// Handle case we are remote NSM and asked for particular endpoint to connect to.
	targetEndpoint := requestConnection.GetNetworkServiceEndpointName()
	if len(targetEndpoint) > 0 {
		endpoint := nsem.model.GetEndpoint(targetEndpoint)
		if endpoint != nil && ignore_endpoints[endpoint.EndpointName()] == nil {
			return endpoint.Endpoint, nil
		} else {
			return nil, fmt.Errorf("Could not find endpoint with name: %s at local registry", targetEndpoint)
		}
	}

	// Get endpoints, do it every time since we do not know if list are changed or not.
	discoveryClient, err := nsem.serviceRegistry.DiscoveryClient()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: requestConnection.GetNetworkService(),
	}
	endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	endpoints := filterEndpoints(endpointResponse.GetNetworkServiceEndpoints(), ignore_endpoints)

	if len(endpoints) == 0 {
		return nil, fmt.Errorf("Failed to find NSE for NetworkService %s. Checked: %d of total NSEs: %d",
			requestConnection.GetNetworkService(), len(ignore_endpoints), len(endpoints))
	}

	endpoint := nsem.model.GetSelector().SelectEndpoint(requestConnection.(*local_connection.Connection), endpointResponse.GetNetworkService(), endpoints)
	if endpoint == nil {
		return nil, err
	}
	return &registry.NSERegistration{
		NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[endpoint.GetNetworkServiceManagerName()],
		NetworkserviceEndpoint: endpoint,
		NetworkService:         endpointResponse.GetNetworkService(),
	}, nil
}

/**
ctx - we assume it is big enought to perform connection.
*/
func (nsem *nseManager) createNSEClient(ctx context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceClient, error) {
	if nsem.isLocalEndpoint(endpoint) {
		modelEp := nsem.model.GetEndpoint(endpoint.GetNetworkserviceEndpoint().GetEndpointName())
		if modelEp == nil {
			return nil, fmt.Errorf("Endpoint not found: %v", endpoint)
		}
		logrus.Infof("Create local NSE connection to endpoint: %v", modelEp)
		client, conn, err := nsem.serviceRegistry.EndpointConnection(ctx, modelEp)
		if err != nil {
			// We failed to connect to local NSE.
			nsem.cleanupNSE(modelEp)
			return nil, err
		}
		return &endpointClient{connection: conn, client: client}, nil
	} else {
		logrus.Infof("Create remote NSE connection to endpoint: %v", endpoint)
		client, conn, err := nsem.serviceRegistry.RemoteNetworkServiceClient(ctx, endpoint.GetNetworkServiceManager())
		if err != nil {
			return nil, err
		}
		return &nsmClient{client: client, connection: conn}, nil
	}
}

func (nsem *nseManager) isLocalEndpoint(endpoint *registry.NSERegistration) bool {
	return nsem.model.GetNsm().GetName() == endpoint.GetNetworkserviceEndpoint().GetNetworkServiceManagerName()
}

func (nsem *nseManager) checkUpdateNSE(ctx context.Context, clientConnection *model.ClientConnection, ep *registry.NetworkServiceEndpoint, endpointResponse *registry.FindNetworkServiceResponse) bool {
	pingCtx, pingCancel := context.WithTimeout(ctx, nsem.properties.HealRequestConnectCheckTimeout)
	defer pingCancel()
	reg := &registry.NSERegistration{
		NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[ep.GetNetworkServiceManagerName()],
		NetworkserviceEndpoint: ep,
		NetworkService:         endpointResponse.GetNetworkService(),
	}

	client, err := nsem.createNSEClient(pingCtx, reg)
	if err == nil && client != nil {
		_ = client.Cleanup()

		// Update endpoint to connect new one.
		clientConnection.Endpoint = reg
		return true
	}
	return false
}

func (nsem *nseManager) cleanupNSE(endpoint *model.Endpoint) {
	// Remove endpoint from model and put workspace into BAD state.
	_ = nsem.model.DeleteEndpoint(endpoint.EndpointName())
	logrus.Infof("NSM: Remove Endpoint since it is not available... %v", endpoint)
}
