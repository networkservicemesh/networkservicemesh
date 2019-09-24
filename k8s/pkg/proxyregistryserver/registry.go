package proxyregistryserver

import (
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/clusterinfo"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"os"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

const (
	NSRegistryForwarderLogPrefix = "Network Service Registry Forwarder"
	NSMRSAddressEnv = "NSMRS_ADDRESS"
)

type nseRegistryService struct {
	clusterInfoService clusterinfo.ClusterInfoServer
}

func newNseRegistryService(clusterInfoService clusterinfo.ClusterInfoServer) *nseRegistryService {
	return &nseRegistryService{
		clusterInfoService: clusterInfoService,
	}
}

func (rs *nseRegistryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("%s: received RegisterNSE(%v)", NSRegistryForwarderLogPrefix, request)

	nsmrsURL := os.Getenv(NSMRSAddressEnv)
	if strings.TrimSpace(nsmrsURL) == "" {
		err := fmt.Errorf("NSMRS Address variable was not set")
		logrus.Warnf("%s: Skipping Register NSE forwarding: %v", NSRegistryForwarderLogPrefix, err)
		return request, err
	}

	nodeConfiguration, cErr := rs.clusterInfoService.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{NodeName: request.NetworkServiceManager.Name})
	if cErr != nil {
		err := fmt.Errorf("cannot get Network Service Manager's IP address: %s", cErr)
		logrus.Errorf("%s: %v", NSRegistryForwarderLogPrefix, err)
		return nil, err
	}

	externalIP := nodeConfiguration.ExternalIP
	if externalIP == "" {
		externalIP = nodeConfiguration.InternalIP
	}
	// Swapping IP address to external (keep port)
	url := request.NetworkServiceManager.Url
	if idx := strings.Index(url, ":"); idx > -1 {
		externalIP += url[idx:]
	}
	request.NetworkServiceManager.Url = externalIP

	logrus.Infof("%s: Prepared forwarding RegisterNSE request: %v", NSRegistryForwarderLogPrefix, request)

	remoteRegistry := nsmd.NewServiceRegistryAt(nsmrsURL + ":80")
	defer remoteRegistry.Stop()

	nseRegistryClient, err := remoteRegistry.NseRegistryClient()
	if err != nil {
		logrus.Warnf(fmt.Sprintf("%s: Cannot register network service endpoint in NSMRS: %v", NSRegistryForwarderLogPrefix, err))
		return request, err
	}

	_, err = nseRegistryClient.RegisterNSE(ctx, request)
	if err != nil {
		errIn := fmt.Errorf("failed register NSE in NSMRS: %v", err)
		logrus.Errorf("%s: %v", NSRegistryForwarderLogPrefix, errIn)
		return request, errIn
	}

	return request, nil
}

func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	logrus.Infof("%s: Received RemoveNSE(%v)", NSRegistryForwarderLogPrefix, request)

	nsmrsURL := os.Getenv(NSMRSAddressEnv)
	if strings.TrimSpace(nsmrsURL) == "" {
		err := fmt.Errorf("NSMRS Address variable was not set")
		logrus.Warnf("%s: Skipping Register NSE forwarding: %v", NSRegistryForwarderLogPrefix, err)
		return &empty.Empty{}, err
	}

	remoteRegistry := nsmd.NewServiceRegistryAt(nsmrsURL + ":80")
	defer remoteRegistry.Stop()

	nseRegistryClient, err := remoteRegistry.NseRegistryClient()
	if err != nil {
		logrus.Warnf(fmt.Sprintf("%s: Cannot register network service endpoint in NSMRS: %v", NSRegistryForwarderLogPrefix, err))
		return &empty.Empty{}, err
	}

	if _, requestErr := nseRegistryClient.RemoveNSE(ctx, request); requestErr != nil {
		return &empty.Empty{}, requestErr
	}

	return &empty.Empty{}, nil
}
