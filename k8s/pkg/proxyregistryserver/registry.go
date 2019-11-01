// Package proxyregistryserver forwarding NSMD k8s commands to remote domain
package proxyregistryserver

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/clusterinfo"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

const (
	// NSRegistryForwarderLogPrefix - log prefix
	NSRegistryForwarderLogPrefix = "Network Service Registry Forwarder"
	// NSMRSAddressEnv - environment variable - address of Network Service Registry Server to forward NSE registry requests
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
		err := errors.Errorf("NSMRS Address variable was not set")
		logrus.Warnf("%s: Skipping Register NSE forwarding: %v", NSRegistryForwarderLogPrefix, err)
		return request, err
	}

	nodeConfiguration, cErr := rs.clusterInfoService.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{NodeName: request.NetworkServiceManager.Name})
	if cErr != nil {
		err := errors.Errorf("cannot get Network Service Manager's IP address: %s", cErr)
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

	nseRegistryClient, err := remoteRegistry.NseRegistryClient(context.Background())
	if err != nil {
		logrus.Warnf(fmt.Sprintf("%s: Cannot register network service endpoint in NSMRS: %v", NSRegistryForwarderLogPrefix, err))
		return request, err
	}

	_, err = nseRegistryClient.RegisterNSE(ctx, request)
	if err != nil {
		errIn := errors.Errorf("failed register NSE in NSMRS: %v", err)
		logrus.Errorf("%s: %v", NSRegistryForwarderLogPrefix, errIn)
		return request, errIn
	}

	return request, nil
}

func (rs *nseRegistryService) BulkRegisterNSE(srv registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	logrus.Infof("Forwarding Bulk Register NSE stream...")

	nsmrsURL := os.Getenv(NSMRSAddressEnv)
	if strings.TrimSpace(nsmrsURL) == "" {
		err := errors.Errorf("NSMRS Address variable was not set")
		logrus.Warnf("%s: Skipping Bulk Register NSE forwarding: %v", NSRegistryForwarderLogPrefix, err)
		return err
	}

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	remoteRegistry := nsmd.NewServiceRegistryAt(nsmrsURL + ":80")
	defer remoteRegistry.Stop()

	nseRegistryClient, err := remoteRegistry.NseRegistryClient(ctx)
	if err != nil {
		err = errors.Errorf("error forwarding BulkRegisterNSE request to %s : %v", nsmrsURL, err)
		return err
	}

	stream, err := nseRegistryClient.BulkRegisterNSE(ctx)
	if err != nil {
		err = errors.Errorf("error forwarding BulkRegisterNSE request to %s : %v", nsmrsURL, err)
		return err
	}

	for {
		request, err := srv.Recv()
		if err != nil {
			err = errors.Errorf("error receiving BulkRegisterNSE request : %v", err)
			return err
		}

		logrus.Infof("Forward BulkRegisterNSE request: %v", request)
		err = stream.Send(request)
		if err != nil {
			err = errors.Errorf("error forwarding BulkRegisterNSE request to %s : %v", nsmrsURL, err)
			return err
		}
	}
}

func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	logrus.Infof("%s: Received RemoveNSE(%v)", NSRegistryForwarderLogPrefix, request)

	nsmrsURL := os.Getenv(NSMRSAddressEnv)
	if strings.TrimSpace(nsmrsURL) == "" {
		err := errors.Errorf("NSMRS Address variable was not set")
		logrus.Warnf("%s: Skipping Register NSE forwarding: %v", NSRegistryForwarderLogPrefix, err)
		return &empty.Empty{}, err
	}

	remoteRegistry := nsmd.NewServiceRegistryAt(nsmrsURL + ":80")
	defer remoteRegistry.Stop()

	nseRegistryClient, err := remoteRegistry.NseRegistryClient(context.Background())
	if err != nil {
		logrus.Warnf(fmt.Sprintf("%s: Cannot register network service endpoint in NSMRS: %v", NSRegistryForwarderLogPrefix, err))
		return &empty.Empty{}, err
	}

	if _, requestErr := nseRegistryClient.RemoveNSE(ctx, request); requestErr != nil {
		return &empty.Empty{}, requestErr
	}

	return &empty.Empty{}, nil
}
