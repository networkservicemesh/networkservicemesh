package proxyregistryserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	utils "github.com/networkservicemesh/networkservicemesh/utils/interdomain"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/clusterinfo"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver"
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
	nodeName           string
}

func newDiscoveryService(cache registryserver.RegistryCache, clusterInfoService clusterinfo.ClusterInfoServer) *discoveryService {
	return &discoveryService{
		cache:              cache,
		clusterInfoService: clusterInfoService,
		nodeName:           os.Getenv("NODE_NAME"),
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

		discoveryClient, dErr := remoteRegistry.DiscoveryClient(context.Background())
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
		managers := make(map[string]*registry.NetworkServiceManager)
		for key, nsm := range response.NetworkServiceManagers {
			if url, urlErr := d.currentDomainNSMgrURL(ctx, d.clusterInfoService, nsm.Url); urlErr == nil && nsm.Url == url {
				d.localizeNSMgr(response, nsm, url)
				managers[nsm.Name] = nsm
				continue
			}
			managers[key] = nsm
			nsm.Name = fmt.Sprintf("%s@%s", nsm.Name, nsm.Url)
			nsmURL := os.Getenv(ProxyNsmdAPIAddressEnv)
			if strings.TrimSpace(nsmURL) == "" {
				nsmURL = ProxyNsmdAPIAddressDefaults
			}
			nsm.Url = nsmURL
			response.NetworkService.Name = originNetworkService
		}
		response.NetworkServiceManagers = managers
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

func (d *discoveryService) localizeNSMgr(response *registry.FindNetworkServiceResponse, m *registry.NetworkServiceManager, url string) {
	logrus.Infof("Handle local find case for mgr: %v of response %v, url: %v", m, response, url)
	m.Name = d.nodeName

	normalizedURL := strings.ReplaceAll(url, ":", "_")

	for _, nse := range response.NetworkServiceEndpoints {
		if strings.Contains(nse.NetworkServiceManagerName, normalizedURL) {
			nse.NetworkServiceManagerName = d.nodeName
		}
	}
}

func (d *discoveryService) currentDomainNSMgrURL(ctx context.Context, clusterInfoService clusterinfo.ClusterInfoServer, u string) (string, error) {
	nodeConfiguration, cErr := clusterInfoService.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{NodeName: d.nodeName})
	if cErr != nil {
		err := errors.Wrapf(cErr, "cannot get Network Service Manager's IP address: %s", cErr)
		return "", err
	}

	externalIP := nodeConfiguration.ExternalIP
	if externalIP == "" {
		externalIP = nodeConfiguration.InternalIP
	}

	if idx := strings.Index(u, ":"); idx > -1 {
		externalIP += u[idx:]
	}

	return externalIP, nil
}
