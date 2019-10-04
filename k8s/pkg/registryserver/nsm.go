package registryserver

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type nsmRegistryService struct {
	nsmName string
	cache   RegistryCache
}

func newNsmRegistryService(nsmName string, cache RegistryCache) *nsmRegistryService {
	return &nsmRegistryService{
		nsmName: nsmName,
		cache:   cache,
	}
}

func (n *nsmRegistryService) RegisterNSM(ctx context.Context, nsm *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	span := spanhelper.FromContext(ctx, "RegisterNSM")
	defer span.Finish()
	span.LogObject("nsm", nsm)
	nsmCr := mapNsmToCustomResource(nsm)
	nsmCr.SetName(n.nsmName)

	span.LogObject("nsm-cr", nsmCr)

	registeredNsm, err := n.cache.CreateOrUpdateNetworkServiceManager(nsmCr)

	span.LogObject("registered-nsm", registeredNsm)
	if err != nil {
		err = fmt.Errorf("Failed to create or update nsm: %s", err)
		span.LogError(err)
		return nil, err
	}

	nsm = mapNsmFromCustomResource(registeredNsm)
	span.LogObject("response", nsm)
	return nsm, nil
}

func (n *nsmRegistryService) GetEndpoints(context.Context, *empty.Empty) (*registry.NetworkServiceEndpointList, error) {
	logrus.Info("Received GetEndpoints")

	var response []*registry.NetworkServiceEndpoint
	for _, endpoint := range n.cache.GetEndpointsByNsm(n.nsmName) {
		response = append(response, mapNseFromCustomResource(endpoint))
	}

	logrus.Infof("GetEndpoints return: %v", response)
	return &registry.NetworkServiceEndpointList{
		NetworkServiceEndpoints: response,
	}, nil
}
