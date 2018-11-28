package serviceregistry

import (
	"net"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/nsmdapi"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	remote_networkservice "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	dataplaneapi "github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"google.golang.org/grpc"
)

type ApiRegistry interface {
	NewNSMServerListener() (net.Listener, error)
	NewPublicListener() (net.Listener, error)
}

/**
A method to obtain different connectivity mechanism for parts of model
*/
type ServiceRegistry interface {
	GetPublicAPI() string

	NetworkServiceDiscovery() (registry.NetworkServiceDiscoveryClient, error)
	RegistryClient() (registry.NetworkServiceRegistryClient, error)

	Stop()
	NSMDApiClient() (nsmdapi.NSMDClient, *grpc.ClientConn, error)
	DataplaneConnection(dataplane *model.Dataplane) (dataplaneapi.DataplaneClient, *grpc.ClientConn, error)

	EndpointConnection(endpoint *registry.NSERegistration) (networkservice.NetworkServiceClient, *grpc.ClientConn, error)
	RemoteNetworkServiceClient(nsm *registry.NetworkServiceManager) (remote_networkservice.NetworkServiceClient, *grpc.ClientConn, error)

	WaitForDataplaneAvailable(model model.Model)

	WorkspaceName(endpoint *registry.NSERegistration) string
}
