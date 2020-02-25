package nsmd

import (
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsmdapi"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/sid"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/vni"
	forwarderapi "github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	// ServerSock defines the path of NSM client socket
	ServerSock = "/var/lib/networkservicemesh/nsm.io.sock"
	// NsmDevicePluginEnv is the name of the env variable to configure enabled device plugin name
	NsmDevicePluginEnv     = "NSM_DEVICE_PLUGIN"
	registryConnectTimeout = time.Second * 30
)

type apiRegistry struct {
}

func (*apiRegistry) NewPublicListener(nsmdAPIAddress string) (net.Listener, error) {
	return net.Listen("tcp", nsmdAPIAddress)
}

func (*apiRegistry) NewNSMServerListener() (net.Listener, error) {
	logrus.Infof("Starting NSM gRPC server listening on socket: %s", ServerSock)
	if err := tools.SocketCleanup(ServerSock); err != nil {
		return nil, err
	}
	return net.Listen("unix", ServerSock)
}

func NewApiRegistry() serviceregistry.ApiRegistry {
	return &apiRegistry{}
}

type nsmdServiceRegistry struct {
	sync.RWMutex
	registryClientConnection *grpc.ClientConn
	stopRedial               bool
	vniAllocator             vni.VniAllocator
	sidAllocator             sid.Allocator
	registryAddress          string
}

func (impl *nsmdServiceRegistry) NewWorkspaceProvider() serviceregistry.WorkspaceLocationProvider {
	return NewDefaultWorkspaceProvider()
}

func (impl *nsmdServiceRegistry) RemoteNetworkServiceClient(ctx context.Context, nsm *registry.NetworkServiceManager) (networkservice.NetworkServiceClient, *grpc.ClientConn, error) {
	err := tools.WaitForPortAvailable(ctx, "tcp", nsm.GetUrl(), 100*time.Millisecond)
	if err != nil {
		return nil, nil, err
	}

	conn, err := tools.DialContextTCP(ctx, nsm.GetUrl())
	if err != nil {
		logrus.Errorf("Failed to dial Remote Network Service Manager %s at %s: %s", nsm.GetName(), nsm.Url, err)
		return nil, nil, err
	}
	client := networkservice.NewNetworkServiceClient(conn)
	logrus.Infof("Connection with Remote Network Service %s at %s is established", nsm.GetName(), nsm.GetUrl())
	return client, conn, nil
}

func (impl *nsmdServiceRegistry) EndpointConnection(ctx context.Context, endpoint *model.Endpoint) (networkservice.NetworkServiceClient, *grpc.ClientConn, error) {
	nseConn, err := tools.DialContextUnix(ctx, endpoint.SocketLocation)
	if err != nil {
		logrus.Errorf("unable to connect to nse %v", endpoint)
		return nil, nil, err
	}
	client := networkservice.NewNetworkServiceClient(nseConn)

	return client, nseConn, nil
}

func (impl *nsmdServiceRegistry) ForwarderConnection(ctx context.Context, forwarder *model.Forwarder) (forwarderapi.ForwarderClient, *grpc.ClientConn, error) {
	forwarderConn, err := tools.DialContextUnix(ctx, forwarder.SocketLocation)
	if err != nil {
		return nil, nil, err
	}
	dpClient := forwarderapi.NewForwarderClient(forwarderConn)
	return dpClient, forwarderConn, nil
}

func (impl *nsmdServiceRegistry) NSMDApiClient(ctx context.Context) (nsmdapi.NSMDClient, *grpc.ClientConn, error) {
	logrus.Infof("Connecting to nsmd on socket: %s...", ServerSock)
	if _, err := os.Stat(ServerSock); err != nil {
		return nil, nil, err
	}

	conn, err := tools.DialContextUnix(ctx, ServerSock)
	if err != nil {
		return nil, nil, err
	}

	logrus.Info("Requesting nsmd for client connection...")
	return nsmdapi.NewNSMDClient(conn), conn, nil
}

func (impl *nsmdServiceRegistry) NseRegistryClient(ctx context.Context) (registry.NetworkServiceRegistryClient, error) {
	impl.RWMutex.Lock()
	defer impl.RWMutex.Unlock()

	logrus.Info("Requesting NseRegistryClient...")
	ctx, cancel := context.WithTimeout(ctx, registryConnectTimeout)
	defer cancel()
	impl.initRegistryClient(ctx)
	if impl.registryClientConnection != nil {
		return registry.NewNetworkServiceRegistryClient(impl.registryClientConnection), nil
	}
	return nil, errors.New("Connection to Network Registry Server is not available")
}

func (impl *nsmdServiceRegistry) NsmRegistryClient(ctx context.Context) (registry.NsmRegistryClient, error) {
	impl.RWMutex.Lock()
	defer impl.RWMutex.Unlock()
	span := spanhelper.GetSpanHelper(ctx)
	span.Logger().Info("Requesting NsmRegistryClient...")
	ctx, cancel := context.WithTimeout(span.Context(), registryConnectTimeout)
	defer cancel()
	impl.initRegistryClient(ctx)
	if impl.registryClientConnection != nil {
		return registry.NewNsmRegistryClient(impl.registryClientConnection), nil
	}
	return nil, errors.New("Connection to Network Registry Server is not available")
}

func (impl *nsmdServiceRegistry) GetPublicAPI() string {
	return GetLocalIPAddress() + ":5001"
}

func (impl *nsmdServiceRegistry) DiscoveryClient(ctx context.Context) (registry.NetworkServiceDiscoveryClient, error) {
	impl.RWMutex.Lock()
	defer impl.RWMutex.Unlock()

	logrus.Info("Requesting NetworkServiceDiscoveryClient...")
	ctx, cancel := context.WithTimeout(ctx, registryConnectTimeout)
	defer cancel()
	impl.initRegistryClient(ctx)
	if impl.registryClientConnection != nil {
		return registry.NewNetworkServiceDiscoveryClient(impl.registryClientConnection), nil
	}
	return nil, errors.New("Connection to Network Registry Server is not available")
}

func (impl *nsmdServiceRegistry) initRegistryClient(ctx context.Context) {
	if impl.registryClientConnection != nil && impl.registryClientConnection.GetState() == connectivity.Ready {
		return // Connection already established.
	}

	span := spanhelper.FromContext(ctx, "initRegistryClient")
	defer span.Finish()

	// TODO doing registry Address here is ugly
	for impl.stopRedial {
		if ctx.Err() != nil {
			return
		}
		err := tools.WaitForPortAvailable(span.Context(), "tcp", impl.registryAddress, 100*time.Millisecond)
		if err != nil {
			err = errors.Wrapf(err, "failed to dial Network Service Registry at %s", impl.registryAddress)
			span.LogError(err)
			continue
		}
		span.Logger().Println("Registry port now available, attempting to connect...")

		conn, err := tools.DialContextTCP(span.Context(), impl.registryAddress)
		if err != nil {
			span.Logger().Errorf("Failed to dial Network Service Registry at %s: %s", impl.registryAddress, err)
			continue
		}
		impl.registryClientConnection = conn
		span.Logger().Infof("Successfully connected to %s", impl.registryAddress)
		return
	}
	span.Logger().Error("stopped before success trying to dial Network Registry Server")
}

func (impl *nsmdServiceRegistry) Stop() {
	// I know the stopRedial isn't threadsafe... we don't care, its set once at creation to true
	// so if you set it to false, eventually the redial loop will notice and stop.
	impl.stopRedial = false
	impl.RWMutex.Lock()
	defer impl.RWMutex.Unlock()

	if impl.registryClientConnection != nil {
		impl.registryClientConnection.Close()
	}
}

func NewServiceRegistry() serviceregistry.ServiceRegistry {
	registryAddress := os.Getenv("NSM_REGISTRY_ADDRESS")
	registryAddress = strings.TrimSpace(registryAddress)
	if registryAddress == "" {
		registryAddress = "127.0.0.1:5000"
	}

	return NewServiceRegistryAt(registryAddress)
}

func NewServiceRegistryAt(nsmAddress string) serviceregistry.ServiceRegistry {
	return &nsmdServiceRegistry{
		stopRedial:      true,
		vniAllocator:    vni.NewVniAllocator(),
		sidAllocator:    sid.NewSIDAllocator(),
		registryAddress: nsmAddress,
	}
}

func (impl *nsmdServiceRegistry) WaitForForwarderAvailable(ctx context.Context, mdl model.Model, timeout time.Duration) error {
	span := spanhelper.FromContext(ctx, "wait-forwarder")
	defer span.Finish()
	span.Logger().Info("Waiting for forwarder available...")

	st := time.Now()
	checkConfigured := func(dp *model.Forwarder) bool {
		return dp.MechanismsConfigured
	}
	for ; true; <-time.After(100 * time.Millisecond) {
		if dp, _ := mdl.SelectForwarder(checkConfigured); dp != nil {
			// We have configured monitor
			return nil
		}
		if time.Since(st) > timeout {
			err := errors.New("error waiting for forwarder... timeout happened")
			span.LogError(err)
		}
	}
	return nil
}

func (impl *nsmdServiceRegistry) VniAllocator() vni.VniAllocator {
	return impl.vniAllocator
}

func (impl *nsmdServiceRegistry) SIDAllocator() sid.Allocator {
	return impl.sidAllocator
}

type defaultWorkspaceProvider struct {
	hostBaseDir     string
	nsmBaseDir      string
	clientBaseDir   string
	nsmServerSocket string
	nsmClientSocket string
	nseRegistryFile string
}

func NewDefaultWorkspaceProvider() serviceregistry.WorkspaceLocationProvider {
	return NewWorkspaceProvider("/var/lib/networkservicemesh/")
}

func NewWorkspaceProvider(rootDir string) serviceregistry.WorkspaceLocationProvider {
	if rootDir[len(rootDir)-1:] != "/" {
		rootDir += "/"
	}
	return &defaultWorkspaceProvider{
		hostBaseDir:     rootDir,
		nsmBaseDir:      rootDir,
		clientBaseDir:   rootDir,
		nsmServerSocket: "nsm.server.io.sock",
		nsmClientSocket: "nsm.client.io.sock",
		nseRegistryFile: "nse.registry",
	}
}

func (w *defaultWorkspaceProvider) HostBaseDir() string {
	return w.hostBaseDir
}

func (w *defaultWorkspaceProvider) NsmBaseDir() string {
	return w.nsmBaseDir
}

func (w *defaultWorkspaceProvider) NsmNSERegistryFile() string {
	return w.nsmBaseDir + w.nseRegistryFile
}

func (w *defaultWorkspaceProvider) ClientBaseDir() string {
	return w.clientBaseDir
}

func (w *defaultWorkspaceProvider) NsmServerSocket() string {
	return w.nsmServerSocket
}

func (w *defaultWorkspaceProvider) NsmClientSocket() string {
	return w.nsmClientSocket
}
