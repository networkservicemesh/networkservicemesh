package nsm

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

type networkServiceEndpointManager interface {
	getEndpoint(ctx context.Context, requestConnection connection.Connection, ignoreEndpoints map[string]*registry.NSERegistration) (*registry.NSERegistration, error)
	createNSEClient(ctx context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceClient, error)
	isLocalEndpoint(endpoint *registry.NSERegistration) bool
	checkUpdateNSE(ctx context.Context, reg *registry.NSERegistration) bool
}

type nseManager struct {
	serviceRegistry serviceregistry.ServiceRegistry
	pluginRegistry  plugins.PluginRegistry
	model           model.Model
	properties      *nsm.NsmProperties
}

func (nsem *nseManager) getEndpoint(ctx context.Context, requestConnection connection.Connection, ignoreEndpoints map[string]*registry.NSERegistration) (*registry.NSERegistration, error) {

	// Handle case we are remote NSM and asked for particular endpoint to connect to.
	targetEndpoint := requestConnection.GetNetworkServiceEndpointName()
	if len(targetEndpoint) > 0 {
		endpoint := nsem.model.GetEndpoint(targetEndpoint)
		if endpoint != nil && ignoreEndpoints[endpoint.EndpointName()] == nil {
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
	endpoints := nsem.filterEndpoints(endpointResponse.GetNetworkServiceEndpoints(), ignoreEndpoints)

	if len(endpoints) == 0 {
		return nil, fmt.Errorf("failed to find NSE for NetworkService %s. Checked: %d of total NSEs: %d",
			requestConnection.GetNetworkService(), len(ignoreEndpoints), len(endpoints))
	}

	endpoint := nsem.model.GetSelector().SelectEndpoint(requestConnection.(*local.Connection), endpointResponse.GetNetworkService(), endpoints)
	if endpoint == nil {
		return nil, fmt.Errorf("failed to find NSE for NetworkService %s. Checked: %d of total NSEs: %d",
			requestConnection.GetNetworkService(), len(ignoreEndpoints), len(endpoints))
	}

	return &registry.NSERegistration{
		NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[endpoint.GetNetworkServiceManagerName()],
		NetworkServiceEndpoint: endpoint,
		NetworkService:         endpointResponse.GetNetworkService(),
	}, nil
}

/**
ctx - we assume it is big enought to perform connection.
*/
func (nsem *nseManager) createNSEClient(ctx context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceClient, error) {
	if nsem.isLocalEndpoint(endpoint) {
		modelEp := nsem.model.GetEndpoint(endpoint.GetNetworkServiceEndpoint().GetName())
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
	return nsem.model.GetNsm().GetName() == endpoint.GetNetworkServiceEndpoint().GetNetworkServiceManagerName()
}

func (nsem *nseManager) checkUpdateNSE(ctx context.Context, reg *registry.NSERegistration) bool {
	pingCtx, pingCancel := context.WithTimeout(ctx, nsem.properties.HealRequestConnectCheckTimeout)
	defer pingCancel()

	client, err := nsem.createNSEClient(pingCtx, reg)
	if err == nil && client != nil {
		_ = client.Cleanup()
		return true
	}

	return false
}

func (nsem *nseManager) cleanupNSE(endpoint *model.Endpoint) {
	// Remove endpoint from model and put workspace into BAD state.
	nsem.model.DeleteEndpoint(endpoint.EndpointName())
	logrus.Infof("NSM: Remove Endpoint since it is not available... %v", endpoint)
}

func (nsem *nseManager) filterEndpoints(endpoints []*registry.NetworkServiceEndpoint, ignoreEndpoints map[string]*registry.NSERegistration) []*registry.NetworkServiceEndpoint {
	result := []*registry.NetworkServiceEndpoint{}
	// Do filter of endpoints
	for _, candidate := range endpoints {
		if ignoreEndpoints[candidate.GetName()] == nil {
			result = append(result, candidate)
		}
	}
	return result
}
