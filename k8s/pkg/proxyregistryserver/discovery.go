package proxyregistryserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/clusterinfo"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/utils"
)

// Default values and environment variables of proxy connection
const (
	ProxyNsmdAPIAddressEnv         = "PROXY_NSMD_ADDRESS"
	ProxyNsmdAPIAddressDefaults    = "pnsmgr-svc:5006"
	ProxyNsmdK8sRemotePortEnv      = "PROXY_NSMD_K8S_REMOTE_PORT"
	ProxyNsmdK8sRemotePortDefaults = "80"
)

type discoveryService struct {
	cache              registryserver.RegistryCache
	clusterInfoService clusterinfo.ClusterInfoServer
}

func newDiscoveryService(cache registryserver.RegistryCache, clusterInfoService clusterinfo.ClusterInfoServer) *discoveryService {
	return &discoveryService{
		cache:              cache,
		clusterInfoService: clusterInfoService,
	}
}

func (d *discoveryService) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {

	networkService, remoteDomain, err := utils.ParseNsmURL(request.NetworkServiceName)
	if err == nil {
		originNetworkService := request.NetworkServiceName

		remoteDomain, err = utils.ResolveDomain(remoteDomain)
		if err != nil {
			return nil, err
		}

		remoteNsrPort := os.Getenv(ProxyNsmdK8sRemotePortEnv)
		if strings.TrimSpace(remoteNsrPort) == "" {
			remoteNsrPort = ProxyNsmdK8sRemotePortDefaults
		}
		remoteRegistry := nsmd.NewServiceRegistryAt(remoteDomain + ":" + remoteNsrPort)
		defer remoteRegistry.Stop()

		discoveryClient, dErr := remoteRegistry.DiscoveryClient()
		if dErr != nil {
			logrus.Error(dErr)
			return nil, dErr
		}

		request.NetworkServiceName = networkService

		logrus.Infof("Transfer request to %v: %v", remoteDomain, request)
		response, dErr := discoveryClient.FindNetworkService(ctx, request)
		if dErr != nil {
			return nil, dErr
		}

		for _, nsm := range response.NetworkServiceManagers {
			nsm.Name = fmt.Sprintf("%s@%s", nsm.Name, nsm.Url)
			nsmURL := os.Getenv(ProxyNsmdAPIAddressEnv)
			if strings.TrimSpace(nsmURL) == "" {
				nsmURL = ProxyNsmdAPIAddressDefaults
			}
			nsm.Url = nsmURL
		}
		response.NetworkService.Name = originNetworkService

		logrus.Infof("Received response: %v", response)
		return response, nil
	}

	response, err := registryserver.FindNetworkServiceWithCache(d.cache, request.NetworkServiceName)
	if err != nil {
		return response, err
	}

	// Swap NSMs IP to external
	for nsmName := range response.NetworkServiceManagers {
		nodeConfiguration, cErr := d.clusterInfoService.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{NodeName: nsmName})
		if cErr != nil {
			logrus.Warnf("Cannot swap Network Service Manager's IP address: %s", cErr)
			continue
		}

		externalIP := nodeConfiguration.ExternalIP
		if externalIP == "" {
			externalIP = nodeConfiguration.InternalIP
		}

		// Swapping IP address to external (keep port)
		url := response.NetworkServiceManagers[nsmName].Url
		if idx := strings.Index(url, ":"); idx > -1 {
			externalIP += url[idx:]
		}
		response.NetworkServiceManagers[nsmName].Url = externalIP
	}
	return response, err
}
