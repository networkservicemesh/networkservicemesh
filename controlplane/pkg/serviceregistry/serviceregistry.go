package serviceregistry

import (
	"net"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsmdapi"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/vni"
	dataplaneapi "github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
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

	DiscoveryClient() (registry.NetworkServiceDiscoveryClient, error)
	NseRegistryClient() (registry.NetworkServiceRegistryClient, error)
	NsmRegistryClient() (registry.NsmRegistryClient, error)

	Stop()
	NSMDApiClient() (nsmdapi.NSMDClient, *grpc.ClientConn, error)
	DataplaneConnection(dataplane *model.Dataplane) (dataplaneapi.DataplaneClient, *grpc.ClientConn, error)

	EndpointConnection(endpoint *registry.NSERegistration) (networkservice.NetworkServiceClient, *grpc.ClientConn, error)
	RemoteNetworkServiceClient(nsm *registry.NetworkServiceManager) (remote_networkservice.NetworkServiceClient, *grpc.ClientConn, error)

	WaitForDataplaneAvailable(model model.Model, timeout time.Duration) error

	WorkspaceName(endpoint *registry.NSERegistration) string

	VniAllocator() vni.VniAllocator

	NewWorkspaceProvider() WorkspaceLocationProvider
}

type WorkspaceLocationProvider interface {
	HostBaseDir() string
	NsmBaseDir() string
	ClientBaseDir() string
	NsmServerSocket() string
	NsmClientSocket() string
}
