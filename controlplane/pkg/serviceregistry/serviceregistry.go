package serviceregistry

import (
	"net"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsmdapi"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/vni"
	forwarderapi "github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

type ApiRegistry interface {
	NewNSMServerListener() (net.Listener, error)
	NewPublicListener(nsmdAPIAddress string) (net.Listener, error)
}

/**
A method to obtain different connectivity mechanism for parts of model
*/
type ServiceRegistry interface {
	GetPublicAPI() string

	DiscoveryClient(ctx context.Context) (registry.NetworkServiceDiscoveryClient, error)
	NseRegistryClient(ctx context.Context) (registry.NetworkServiceRegistryClient, error)
	NsmRegistryClient(ctx context.Context) (registry.NsmRegistryClient, error)

	Stop()
	NSMDApiClient(ctx context.Context) (nsmdapi.NSMDClient, *grpc.ClientConn, error)
	ForwarderConnection(ctx context.Context, forwarder *model.Forwarder) (forwarderapi.ForwarderClient, *grpc.ClientConn, error)

	EndpointConnection(ctx context.Context, endpoint *model.Endpoint) (networkservice.NetworkServiceClient, *grpc.ClientConn, error)
	RemoteNetworkServiceClient(ctx context.Context, nsm *registry.NetworkServiceManager) (networkservice.NetworkServiceClient, *grpc.ClientConn, error)

	WaitForForwarderAvailable(ctx context.Context, model model.Model, timeout time.Duration) error

	VniAllocator() vni.VniAllocator

	NewWorkspaceProvider() WorkspaceLocationProvider
}

type WorkspaceLocationProvider interface {
	HostBaseDir() string
	NsmBaseDir() string
	ClientBaseDir() string
	NsmServerSocket() string
	NsmClientSocket() string

	// A persistent file based NSE <-> Workspace registry.
	NsmNSERegistryFile() string
}
