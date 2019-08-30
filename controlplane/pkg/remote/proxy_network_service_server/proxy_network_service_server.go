package proxynetworkserviceserver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/clusterinfo"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/utils"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

// Default values and environment variables of proxy connection
const (
	ProxyNsmdK8sAddressEnv         = "PROXY_NSMD_K8S_ADDRESS"
	ProxyNsmdK8sAddressDefaults    = "pnsmgr-svc:5005"
	ProxyNsmdK8sRemotePortEnv      = "PROXY_NSMD_K8S_REMOTE_PORT"
	ProxyNsmdK8sRemotePortDefaults = "80"

	RequestConnectTimeout  = 15 * time.Second
	RequestConnectAttempts = 3
)

type proxyNetworkServiceServer struct {
	serviceRegistry serviceregistry.ServiceRegistry
}

// NewProxyNetworkServiceServer creates a new remote.NetworkServiceServer
func NewProxyNetworkServiceServer(serviceRegistry serviceregistry.ServiceRegistry) remote_networkservice.NetworkServiceServer {
	server := &proxyNetworkServiceServer{
		serviceRegistry: serviceRegistry,
	}
	return server
}

func (srv *proxyNetworkServiceServer) Request(ctx context.Context, request *remote_networkservice.NetworkServiceRequest) (*remote_connection.Connection, error) {
	logrus.Infof("ProxyNSMD: Received request from client to connect to NetworkService: %v", request)

	destNsmName := request.Connection.DestinationNetworkServiceManagerName
	dNsmName, dNsmAddress, err := utils.ParseNsmURL(destNsmName)
	if err != nil {
		return nil, fmt.Errorf("ProxyNSMD: Failed to extract destination nsm address")
	}

	request.Connection.DestinationNetworkServiceManagerName = dNsmName

	dNsm := &registry.NetworkServiceManager{
		Name: dNsmName,
		Url:  dNsmAddress,
	}

	var client remote_networkservice.NetworkServiceClient
	var conn *grpc.ClientConn
	for i := 0; i < RequestConnectAttempts; i++ {
		rnsCtx, pingCancel := context.WithTimeout(ctx, RequestConnectTimeout)
		defer pingCancel()

		client, conn, err = srv.serviceRegistry.RemoteNetworkServiceClient(rnsCtx, dNsm)
		if err == nil {
			break
		}
	}
	if err != nil {
		logrus.Errorf("ProxyNSMD: Failed connect to Network Service Client (%s): %v", destNsmName, err)
		return nil, err
	}

	defer func() {
		if e := conn.Close(); e != nil {
			logrus.Errorf("ProxyNSMD: Failed to close Network Service Client (%s): %v", destNsmName, e)
		}
	}()

	localNsrURL := os.Getenv(ProxyNsmdK8sAddressEnv)
	if strings.TrimSpace(localNsrURL) == "" {
		localNsrURL = ProxyNsmdK8sAddressDefaults
	}

	localClusterInfoClient, localConn, err := createClusterInfoClient(ctx, localNsrURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = localConn.Close(); err != nil {
			logrus.Errorf("ProxyNSMD: Failed to close the local Cluster Info Client (%s). %v", localNsrURL, err)
		}
	}()

	remoteNsrPort := os.Getenv(ProxyNsmdK8sRemotePortEnv)
	if strings.TrimSpace(remoteNsrPort) == "" {
		remoteNsrPort = ProxyNsmdK8sRemotePortDefaults
	}

	remoteRegistryAddress := dNsmAddress[:strings.Index(dNsmAddress, ":")] + ":" + remoteNsrPort
	logrus.Infof("ProxyNSMD: Connecting to remote service registry at %v", remoteRegistryAddress)

	remoteClusterInfoClient, remoteConn, err := createClusterInfoClient(ctx, remoteRegistryAddress)
	if err != nil {
		logrus.Errorf("ProxyNSMD: Failed connecting to remote service registry at %v: %v", remoteRegistryAddress, err)
		return nil, err
	}
	defer func() {
		if err = remoteConn.Close(); err != nil {
			logrus.Errorf("ProxyNSMD: Failed to close the remote Cluster Info Client (%s). %v", remoteRegistryAddress, err)
		}
	}()

	localSrcIP := request.MechanismPreferences[0].Parameters["src_ip"]

	localNodeIPConfiguration, err := localClusterInfoClient.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{InternalIP: localSrcIP})
	if err == nil {
		if len(localNodeIPConfiguration.ExternalIP) > 0 {
			request.MechanismPreferences[0].Parameters["src_ip"] = localNodeIPConfiguration.ExternalIP
		}
	}

	logrus.Infof("ProxyNSMD: Sending request to remote network service: %v", request)

	response, err := client.Request(ctx, request)

	if err != nil {
		return response, err
	}

	remoteNodeIPConfiguration, err := remoteClusterInfoClient.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{InternalIP: response.Mechanism.Parameters["dst_ip"]})
	if err == nil {
		if len(remoteNodeIPConfiguration.ExternalIP) > 0 {
			response.Mechanism.Parameters["dst_ip"] = remoteNodeIPConfiguration.ExternalIP
		}
	}

	response.Mechanism.Parameters["src_ip"] = localSrcIP
	response.DestinationNetworkServiceManagerName = destNsmName

	logrus.Infof("ProxyNSMD: Received response from remote network service: %v", response)

	return response, err
}

func (srv *proxyNetworkServiceServer) Close(ctx context.Context, connection *remote_connection.Connection) (*empty.Empty, error) {
	logrus.Infof("ProxyNSMD: Proxy closing connection: %v", *connection)

	destNsmName := connection.DestinationNetworkServiceManagerName
	dNsmName, dNsmAddress, err := utils.ParseNsmURL(destNsmName)
	if err != nil {
		return nil, fmt.Errorf("ProxyNSMD: Failed to extract destination nsm address")
	}

	dNsm := &registry.NetworkServiceManager{
		Name: dNsmName,
		Url:  dNsmAddress,
	}

	client, conn, err := srv.serviceRegistry.RemoteNetworkServiceClient(ctx, dNsm)
	if err != nil {
		logrus.Errorf("ProxyNSMD: Failed to create NSE Client. %v", err)
		return nil, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.Errorf("ProxyNSMD: Failed to close NSE Client. %v", err)
		}
	}()

	return client.Close(ctx, connection)
}

func createClusterInfoClient(ctx context.Context, address string) (clusterinfo.ClusterInfoClient, *grpc.ClientConn, error) {
	err := tools.WaitForPortAvailable(ctx, "tcp", address, 100*time.Millisecond)
	if err != nil {
		return nil, nil, err
	}

	conn, err := tools.DialContextTCP(ctx, address)
	if err != nil {
		return nil, nil, err
	}

	client := clusterinfo.NewClusterInfoClient(conn)
	return client, conn, nil
}
