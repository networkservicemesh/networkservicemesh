package nsm

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	nsm_properties "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"

	"github.com/sirupsen/logrus"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

type nseManager struct {
	serviceRegistry serviceregistry.ServiceRegistry
	model           model.Model
	properties      *nsm_properties.Properties
}

func (nsem *nseManager) GetEndpoint(ctx context.Context, requestConnection connection.Connection, ignoreEndpoints map[string]*registry.NSERegistration) (*registry.NSERegistration, error) {

	span := spanhelper.FromContext(ctx, "GetEndpoint")
	defer span.Finish()
	span.LogObject("request", requestConnection)
	span.LogObject("ignores", ignoreEndpoints)
	// Handle case we are remote NSM and asked for particular endpoint to connect to.
	targetEndpoint := requestConnection.GetNetworkServiceEndpointName()
	span.LogObject("targetEndpoint", targetEndpoint)
	if len(targetEndpoint) > 0 {
		endpoint := nsem.model.GetEndpoint(targetEndpoint)
		if endpoint != nil && ignoreEndpoints[endpoint.EndpointName()] == nil {
			return endpoint.Endpoint, nil
		} else {
			return nil, fmt.Errorf("Could not find endpoint with name: %s at local registry", targetEndpoint)
		}
	}

	// Get endpoints, do it every time since we do not know if list are changed or not.
	discoveryClient, err := nsem.serviceRegistry.DiscoveryClient(ctx)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: requestConnection.GetNetworkService(),
	}
	span.LogObject("nseRequest", nseRequest)
	endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
	span.LogObject("nseResponse", endpointResponse)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	endpoints := nsem.filterEndpoints(endpointResponse.GetNetworkServiceEndpoints(), ignoreEndpoints)

	if len(endpoints) == 0 {
		err = fmt.Errorf("failed to find NSE for NetworkService %s. Checked: %d of total NSEs: %d",
			requestConnection.GetNetworkService(), len(ignoreEndpoints), len(endpoints))
		span.LogError(err)
		return nil, err
	}

	endpoint := nsem.model.GetSelector().SelectEndpoint(requestConnection.(*local.Connection), endpointResponse.GetNetworkService(), endpoints)
	if endpoint == nil {
		err = fmt.Errorf("failed to find NSE for NetworkService %s. Checked: %d of total NSEs: %d",
			requestConnection.GetNetworkService(), len(ignoreEndpoints), len(endpoints))
		span.LogError(err)
		return nil, err
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
func (nsem *nseManager) CreateNSEClient(ctx context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceClient, error) {
	span := spanhelper.GetSpanHelper(ctx)
	logger := span.Logger()

	if nsem.IsLocalEndpoint(endpoint) {
		modelEp := nsem.model.GetEndpoint(endpoint.GetNetworkServiceEndpoint().GetName())
		if modelEp == nil {
			return nil, fmt.Errorf("Endpoint not found: %v", endpoint)
		}
		logger.Infof("Create local NSE connection to endpoint: %v", modelEp)
		client, conn, err := nsem.serviceRegistry.EndpointConnection(ctx, modelEp)
		if err != nil {
			span.LogError(err)
			// We failed to connect to local NSE.
			nsem.cleanupNSE(ctx, modelEp)
			return nil, err
		}
		return &endpointClient{connection: conn, client: client}, nil
	} else {
		logger.Infof("Create remote NSE connection to endpoint: %v", endpoint)
		client, conn, err := nsem.serviceRegistry.RemoteNetworkServiceClient(ctx, endpoint.GetNetworkServiceManager())
		if err != nil {
			return nil, err
		}
		return &nsmClient{client: client, connection: conn}, nil
	}
}

func (nsem *nseManager) IsLocalEndpoint(endpoint *registry.NSERegistration) bool {
	return nsem.model.GetNsm().GetName() == endpoint.GetNetworkServiceEndpoint().GetNetworkServiceManagerName()
}

func (nsem *nseManager) CheckUpdateNSE(ctx context.Context, reg *registry.NSERegistration) bool {
	pingCtx, pingCancel := context.WithTimeout(ctx, nsem.properties.HealRequestConnectCheckTimeout)
	defer pingCancel()

	client, err := nsem.CreateNSEClient(pingCtx, reg)
	if err == nil && client != nil {
		_ = client.Cleanup()
		return true
	}

	return false
}

func (nsem *nseManager) cleanupNSE(ctx context.Context, endpoint *model.Endpoint) {
	// Remove endpoint from model and put workspace into BAD state.
	nsem.model.DeleteEndpoint(ctx, endpoint.EndpointName())
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
