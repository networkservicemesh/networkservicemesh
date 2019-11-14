package proxynetworkserviceserver

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/utils/interdomain"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/clusterinfo"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
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
func NewProxyNetworkServiceServer(serviceRegistry serviceregistry.ServiceRegistry) networkservice.NetworkServiceServer {
	server := &proxyNetworkServiceServer{
		serviceRegistry: serviceRegistry,
	}
	return server
}

func (srv *proxyNetworkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("ProxyNSMD: Received request from client to connect to NetworkService: %v", request)

	destNsmName := request.Connection.GetDestinationNetworkServiceManagerName()
	dNsmName, _, err := interdomain.ParseNsmURL(destNsmName)
	if err != nil {
		return nil, errors.New("ProxyNSMD: Failed to extract destination nsm address")
	}

	// Check point is final
	_, _, err = interdomain.ParseNsmURL(dNsmName)
	if err != nil {
		return srv.request(ctx, request, destNsmName, true)
	}

	return srv.request(ctx, request, destNsmName, false)
}

func (srv *proxyNetworkServiceServer) request(ctx context.Context, request *networkservice.NetworkServiceRequest, destNsmName string, isRemoteSide bool) (*connection.Connection, error) {
	dNsmName, dNsmAddress, err := interdomain.ParseNsmURL(destNsmName)
	if err != nil {
		return nil, errors.New("ProxyNSMD: Failed to extract destination nsm address")
	}

	request.Connection.NetworkServiceManagers[1] = dNsmName

	dNsm := srv.newManager(dNsmName, dNsmAddress)
	client, conn, err := srv.connectNSM(ctx, dNsm)
	if err != nil {
		logrus.Errorf("ProxyNSMD: Failed connect to Network Service Client (%s): %v", dNsmAddress, err)
		return nil, err
	}
	defer func() {
		if e := conn.Close(); e != nil {
			logrus.Errorf("ProxyNSMD: Failed to close Network Service Client (%s): %v", dNsmAddress, e)
		}
	}()
	nsrURL := srv.getLocalNsrURL()
	clusterInfoClient, localConn, err := createClusterInfoClient(ctx, nsrURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = localConn.Close(); err != nil {
			logrus.Errorf("ProxyNSMD: Failed to close the local Cluster Info Client (%s). %v", nsrURL, err)
		}
	}()

	var response *connection.Connection

	if isRemoteSide {
		logrus.Infof("ProxyNSMD: Sending request to remote network service: %v", request)
		response, err = client.Request(ctx, request)
		if err != nil {
			return response, err
		}
		srv.updateResponseRemote(ctx, response, clusterInfoClient)
	} else {
		localSrcIP, originalNetworkService := srv.updateParametersLocal(ctx, request, dNsmAddress, clusterInfoClient)
		logrus.Infof("ProxyNSMD: Sending request to remote network service: %v", request)
		response, err = client.Request(ctx, request)
		if err != nil {
			return response, err
		}
		srv.updateResponseLocal(response, localSrcIP, destNsmName, originalNetworkService)
	}
	logrus.Infof("ProxyNSMD: Received response from remote network service: %v", response)
	return response, err
}

func (srv *proxyNetworkServiceServer) updateParametersLocal(ctx context.Context, request *networkservice.NetworkServiceRequest, dNsmAddress string, localClusterInfoClient clusterinfo.ClusterInfoClient) (string, string) {
	localSrcIP := request.MechanismPreferences[0].Parameters[vxlan.SrcIP]
	request.MechanismPreferences[0].Parameters[vxlan.DstExternalIP] = dNsmAddress[:strings.Index(dNsmAddress, ":")]

	localNodeIPConfiguration, err := localClusterInfoClient.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{InternalIP: localSrcIP})
	if err == nil {
		if len(localNodeIPConfiguration.ExternalIP) > 0 {
			request.MechanismPreferences[0].Parameters[vxlan.SrcIP] = localNodeIPConfiguration.ExternalIP
			request.MechanismPreferences[0].Parameters[vxlan.SrcOriginalIP] = localSrcIP
		}
	}

	originalNetworkService := request.Connection.NetworkService
	var networkService string
	networkService, _, err = interdomain.ParseNsmURL(originalNetworkService)
	if err == nil {
		request.Connection.NetworkService = networkService
	} else {
		logrus.Warnf("Cannot parse Network Service name %s, keep original", originalNetworkService)
	}
	return localSrcIP, originalNetworkService
}

func (srv *proxyNetworkServiceServer) updateResponseLocal(response *connection.Connection, localSrcIP, destNsmName, originalNetworkService string) {
	response.Mechanism.Parameters[vxlan.SrcIP] = localSrcIP
	response.NetworkServiceManagers[1] = destNsmName
	response.NetworkService = originalNetworkService
}

func (srv *proxyNetworkServiceServer) updateResponseRemote(ctx context.Context, response *connection.Connection, clusterInfoClient clusterinfo.ClusterInfoClient) {
	remoteNodeIPConfiguration, err := clusterInfoClient.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{InternalIP: response.Mechanism.Parameters["dst_ip"]})
	if err == nil {
		if len(remoteNodeIPConfiguration.ExternalIP) > 0 {
			response.Mechanism.Parameters[vxlan.DstIP] = remoteNodeIPConfiguration.ExternalIP
		}
	}
}

func (srv *proxyNetworkServiceServer) newManager(dNsmName, dNsmAddress string) *registry.NetworkServiceManager {
	return &registry.NetworkServiceManager{
		Name: dNsmName,
		Url:  dNsmAddress,
	}
}

func (srv *proxyNetworkServiceServer) connectNSM(ctx context.Context, dNsm *registry.NetworkServiceManager) (networkservice.NetworkServiceClient, *grpc.ClientConn, error) {
	var client networkservice.NetworkServiceClient
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < RequestConnectAttempts; i++ {
		rnsCtx, pingCancel := context.WithTimeout(ctx, RequestConnectTimeout)
		defer pingCancel()

		client, conn, err = srv.serviceRegistry.RemoteNetworkServiceClient(rnsCtx, dNsm)
		if err == nil {
			break
		}
	}
	return client, conn, err
}

func (srv *proxyNetworkServiceServer) getLocalNsrURL() string {
	localNsrURL := os.Getenv(ProxyNsmdK8sAddressEnv)
	if strings.TrimSpace(localNsrURL) == "" {
		localNsrURL = ProxyNsmdK8sAddressDefaults
	}
	return localNsrURL
}

func (srv *proxyNetworkServiceServer) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	logrus.Infof("ProxyNSMD: Proxy closing connection: %v", *connection)

	destNsmName := connection.NetworkServiceManagers[1]
	dNsmName, dNsmAddress, err := interdomain.ParseNsmURL(destNsmName)
	if err != nil {
		return nil, errors.Errorf("ProxyNSMD: Failed to extract destination nsm address")
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

	connection.NetworkServiceManagers[1] = dNsmName

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
