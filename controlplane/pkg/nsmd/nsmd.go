package nsmd

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/probes"
	"github.com/networkservicemesh/networkservicemesh/pkg/probes/health"

	"github.com/opentracing/opentracing-go"

	remote_server "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsmdapi"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nseregistry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/sdk/monitor/crossconnect"
)

const (
	NsmdDeleteLocalRegistry = "NSMD_LOCAL_REGISTRY_DELETE"
	DataplaneTimeout        = 1 * time.Hour
	NSEAliveTimeout         = 1 * time.Second
)

type NSMServer interface {
	Stop()
	StartDataplaneRegistratorServer() error
	StartAPIServerAt(ctx context.Context, sock net.Listener, probes probes.Probes)

	XconManager() *services.ClientConnectionManager
	Manager() nsm.NetworkServiceManager

	MonitorManager
	EndpointManager
}

type nsmServer struct {
	sync.Mutex
	workspaces       map[string]*Workspace
	model            model.Model
	serviceRegistry  serviceregistry.ServiceRegistry
	manager          nsm.NetworkServiceManager
	locationProvider serviceregistry.WorkspaceLocationProvider
	localRegistry    *nseregistry.NSERegistry
	registerServer   *grpc.Server
	registerSock     net.Listener
	regServer        *dataplaneRegistrarServer

	xconManager             *services.ClientConnectionManager
	crossConnectMonitor     monitor_crossconnect.MonitorServer
	remoteConnectionMonitor remote.MonitorServer
	remoteServer            remote_networkservice.NetworkServiceServer
}

func (nsm *nsmServer) XconManager() *services.ClientConnectionManager {
	return nsm.xconManager
}

func (nsm *nsmServer) Manager() nsm.NetworkServiceManager {
	return nsm.manager
}

func (nsm *nsmServer) LocalConnectionMonitor(workspace string) monitor.Server {
	if ws := nsm.workspaces[workspace]; ws != nil {
		return ws.MonitorConnectionServer()
	}

	return nil
}

func (nsm *nsmServer) CrossConnectMonitor() monitor_crossconnect.MonitorServer {
	return nsm.crossConnectMonitor
}

func (nsm *nsmServer) RemoteConnectionMonitor() monitor.Server {
	return nsm.remoteConnectionMonitor
}

func RequestWorkspace(serviceRegistry serviceregistry.ServiceRegistry, id string) (*nsmdapi.ClientConnectionReply, error) {
	client, con, err := serviceRegistry.NSMDApiClient()
	if err != nil {
		logrus.Fatalf("Failed to connect to NSMD: %+v...", err)
	}
	defer con.Close()

	reply, err := client.RequestClientConnection(context.Background(), &nsmdapi.ClientConnectionRequest{Workspace: id})
	if err != nil {
		return nil, err
	}
	logrus.Infof("nsmd allocated workspace %+v for client operations...", reply)
	return reply, nil
}

func (nsm *nsmServer) RequestClientConnection(context context.Context, request *nsmdapi.ClientConnectionRequest) (*nsmdapi.ClientConnectionReply, error) {
	logrus.Infof("Requested client connection to nsmd : %+v", request)

	workspace, err := NewWorkSpace(nsm, request.Workspace, false)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logrus.Infof("New workspace created: %+v", workspace)

	err = nsm.localRegistry.AppendClientRequest(workspace.Name())
	if err != nil {
		logrus.Errorf("Failed to store Client information into local registry: %v", err)
		return nil, err
	}
	nsm.Lock()
	nsm.workspaces[workspace.Name()] = workspace
	nsm.Unlock()
	reply := &nsmdapi.ClientConnectionReply{
		Workspace:       workspace.Name(),
		HostBasedir:     workspace.locationProvider.HostBaseDir(),
		ClientBaseDir:   workspace.locationProvider.ClientBaseDir(),
		NsmServerSocket: workspace.locationProvider.NsmServerSocket(),
		NsmClientSocket: workspace.locationProvider.NsmClientSocket(),
	}
	logrus.Infof("returning ClientConnectionReply: %+v", reply)
	return reply, nil
}

func (nsm *nsmServer) DeleteClientConnection(context context.Context, request *nsmdapi.DeleteConnectionRequest) (*nsmdapi.DeleteConnectionReply, error) {
	socket := request.Workspace
	logrus.Infof("Delete connection for workspace %s", socket)

	workspace, ok := nsm.workspaces[socket]
	if !ok {
		err := fmt.Errorf("no connection exists for workspace %s", socket)
		return &nsmdapi.DeleteConnectionReply{}, err
	}

	err := nsm.localRegistry.DeleteClient(workspace.Name())
	if err != nil {
		logrus.Errorf("Failed to delete Client information into local registry: %v", err)
		return nil, err
	}

	workspace.Close()
	nsm.Lock()
	delete(nsm.workspaces, socket)
	nsm.Unlock()

	return &nsmdapi.DeleteConnectionReply{}, nil
}

func (nsm *nsmServer) EnumConnection(context context.Context, request *nsmdapi.EnumConnectionRequest) (*nsmdapi.EnumConnectionReply, error) {
	nsm.Lock()
	defer nsm.Unlock()
	workspaces := []string{}
	for w := range nsm.workspaces {
		if len(w) > 0 {
			workspaces = append(workspaces, w)
		}
	}
	return &nsmdapi.EnumConnectionReply{Workspace: workspaces}, nil
}

func (nsm *nsmServer) restore(registeredEndpointsList *registry.NetworkServiceEndpointList) {
	if os.Getenv(NsmdDeleteLocalRegistry) == "true" {
		logrus.Errorf("Delete of local nse/client registry... by ENV VAR: %s", NsmdDeleteLocalRegistry)
		nsm.localRegistry.Delete()
	}

	clients, nses, err := nsm.localRegistry.LoadRegistry()
	if err != nil {
		logrus.Errorf("NSMServer: Error Loading stored NSE registry: %v", err)
		return
	}

	registeredNSEs := map[string]string{}
	for _, endpoint := range registeredEndpointsList.GetNetworkServiceEndpoints() {
		registeredNSEs[endpoint.GetName()] = endpoint.GetNetworkServiceName()
	}

	updatedClients := nsm.restoreClients(clients)
	updatedEndpoints, err := nsm.restoreEndpoints(nses, registeredNSEs)
	if err != nil {
		logrus.Errorf("Error restoring endpoints: %v", err)
		return
	}

	if len(updatedClients) > 0 || len(updatedEndpoints) > 0 {
		if err := nsm.localRegistry.Save(updatedClients, updatedEndpoints); err != nil {
			logrus.Errorf("Store updated NSE local registry...")
		}
	}

	logrus.Infof("NSMD: Restore of NSE/Clients Complete...")
}

func (nsm *nsmServer) restoreClients(clients []string) []string {
	nsm.Lock()
	defer nsm.Unlock()

	logrus.Infof("NSMServer: Creating workspaces for existing clients...")

	updatedClients := make([]string, 0, len(clients))
	for _, client := range clients {
		if client == "" {
			continue
		}
		workspace, err := NewWorkSpace(nsm, client, true)
		if err != nil {
			logrus.Errorf("NSMServer: Failed to create workspace %s %v. Ignoring...", client, err)
			continue
		}
		nsm.workspaces[workspace.Name()] = workspace
		updatedClients = append(updatedClients, client)
	}

	return updatedClients
}

func (nsm *nsmServer) restoreEndpoints(nses map[string]nseregistry.NSEEntry, registeredNSEs map[string]string) (map[string]nseregistry.NSEEntry, error) {
	discoveryClient, err := nsm.serviceRegistry.DiscoveryClient()
	if err != nil {
		logrus.Errorf("Failed to get DiscoveryClient: %v", err)
		return nil, err
	}

	registryClient, err := nsm.serviceRegistry.NseRegistryClient()
	if err != nil {
		logrus.Errorf("Failed to get RegistryClient: %v", err)
		return nil, err
	}

	networkServices := map[string]bool{}
	updatedNSEs := map[string]nseregistry.NSEEntry{}
	for name, nse := range nses {
		ws, ok := nsm.workspaces[nse.Workspace]
		if !ok {
			continue
		}

		logrus.Infof("Checking NSE %s is alive at %v...", name, ws.NsmClientSocket())
		if !ws.isConnectionAlive(NSEAliveTimeout) {
			logrus.Errorf("Unable to connect to local nse %v. Skipping", nse.NseReg)
			if err := nsm.deleteEndpointWithClient(name, registryClient); err != nil {
				logrus.Errorf("Remove NSE: NSE %v", err)
			}
			continue
		}

		logrus.Infof("NSE %s is alive at %v...", name, ws.NsmClientSocket())
		newName, newNSE, err := nsm.restoreEndpoint(discoveryClient, registryClient, name, nse, ws, registeredNSEs, networkServices)
		if err != nil {
			logrus.Errorf("Failed to register NSE: %v", err)
			continue
		}

		networkServices[newNSE.NseReg.GetNetworkService().GetName()] = true
		updatedNSEs[newName] = newNSE
	}

	// We need to unregister NSE's without NSM registration.
	for name := range registeredNSEs {
		if _, ok := updatedNSEs[name]; !ok {
			if _, err := registryClient.RemoveNSE(context.Background(), &registry.RemoveNSERequest{
				NetworkServiceEndpointName: name,
			}); err != nil {
				logrus.Errorf("Remove NSE: NSE %v", err)
			}
		}
	}

	return updatedNSEs, nil
}

func (nsm *nsmServer) restoreEndpoint(
	discoveryClient registry.NetworkServiceDiscoveryClient,
	registryClient registry.NetworkServiceRegistryClient,
	name string,
	nse nseregistry.NSEEntry,
	ws *Workspace,
	registeredNSEs map[string]string,
	networkServices map[string]bool) (string, nseregistry.NSEEntry, error) {

	if networkService, ok := registeredNSEs[name]; ok {
		if networkServices[networkService] {
			nsm.restoreRegisteredEndpoint(nse, ws)
			return name, nse, nil
		}

		if _, err := discoveryClient.FindNetworkService(context.Background(), &registry.FindNetworkServiceRequest{
			NetworkServiceName: networkService,
		}); err == nil {
			nsm.restoreRegisteredEndpoint(nse, ws)
			return name, nse, nil
		}

		if err := nsm.deleteEndpointWithClient(name, registryClient); err != nil {
			return "", nseregistry.NSEEntry{}, err
		}
	}

	return nsm.restoreNotRegisteredEndpoint(registryClient, nse, ws)
}

func (nsm *nsmServer) restoreRegisteredEndpoint(nse nseregistry.NSEEntry, ws *Workspace) {
	nse.NseReg.NetworkServiceManager = nsm.model.GetNsm()
	nse.NseReg.NetworkServiceEndpoint.NetworkServiceManagerName = nse.NseReg.GetNetworkServiceManager().GetName()

	nsm.model.AddEndpoint(&model.Endpoint{
		Endpoint:       nse.NseReg,
		Workspace:      nse.Workspace,
		SocketLocation: ws.NsmClientSocket(),
	})
}

func (nsm *nsmServer) restoreNotRegisteredEndpoint(
	registryClient registry.NetworkServiceRegistryClient,
	nse nseregistry.NSEEntry,
	ws *Workspace) (string, nseregistry.NSEEntry, error) {

	reg, err := ws.registryServer.RegisterNSEWithClient(context.Background(), nse.NseReg, registryClient)
	if err != nil {
		name := nse.NseReg.GetNetworkServiceEndpoint().GetName()
		logrus.Warnf("Failed to register NSE with name %v: %v", name, err)

		nse.NseReg.NetworkServiceEndpoint.Name = ""
		if reg, err = ws.registryServer.RegisterNSEWithClient(context.Background(), nse.NseReg, registryClient); err != nil {
			return "", nseregistry.NSEEntry{}, err
		}

		nsm.manager.NotifyRenamedEndpoint(name, reg.GetNetworkServiceEndpoint().GetName())
	}

	return reg.GetNetworkServiceEndpoint().GetName(), nseregistry.NSEEntry{
		Workspace: ws.Name(),
		NseReg:    reg,
	}, nil
}

func (nsm *nsmServer) deleteEndpointWithClient(name string, client registry.NetworkServiceRegistryClient) error {
	if _, err := client.RemoveNSE(context.Background(), &registry.RemoveNSERequest{
		NetworkServiceEndpointName: name,
	}); err != nil {
		return err
	}

	nsm.model.DeleteEndpoint(name)

	return nil
}

// DeleteEndpointWithBrokenConnection deletes endpoint if it has no active connections
func (nsm *nsmServer) DeleteEndpointWithBrokenConnection(endpoint *model.Endpoint) error {
	// If endpoint has active client connection, it should be handled by MonitorNetNsInodeServer
	for _, clientConnection := range nsm.model.GetAllClientConnections() {
		if endpoint.EndpointName() == clientConnection.Endpoint.GetNetworkServiceEndpoint().GetName() {
			return nil
		}
	}

	client, err := nsm.serviceRegistry.NseRegistryClient()
	if err != nil {
		return err
	}

	return nsm.deleteEndpointWithClient(endpoint.EndpointName(), client)
}

func (nsm *nsmServer) Stop() {
	if nsm.registerServer != nil {
		nsm.registerServer.GracefulStop()
	}
	if nsm.registerSock != nil {
		_ = nsm.registerSock.Close()
	}
	if nsm.regServer != nil {
		nsm.regServer.Stop()
	}
}

// StartNSMServer registers and starts gRPC server which is listening for
// Network Service requests.
func StartNSMServer(ctx context.Context, model model.Model, manager nsm.NetworkServiceManager, apiRegistry serviceregistry.ApiRegistry) (NSMServer, error) {
	if opentracing.IsGlobalTracerRegistered() {
		span, _ := opentracing.StartSpanFromContext(ctx, "nsm.server.start")
		defer span.Finish()
	}

	var err error
	if err = tools.SocketCleanup(ServerSock); err != nil {
		return nil, err
	}

	locationProvider := manager.ServiceRegistry().NewWorkspaceProvider()

	nsm := &nsmServer{
		workspaces:       make(map[string]*Workspace),
		model:            model,
		serviceRegistry:  manager.ServiceRegistry(),
		manager:          manager,
		locationProvider: locationProvider,
		localRegistry:    nseregistry.NewNSERegistry(locationProvider.NsmNSERegistryFile()),
	}

	nsm.registerServer = tools.NewServer()
	nsmdapi.RegisterNSMDServer(nsm.registerServer, nsm)

	nsm.registerSock, err = apiRegistry.NewNSMServerListener()
	if err != nil {
		logrus.Errorf("failed to start device plugin grpc server %+v", err)
		nsm.Stop()
		return nil, err
	}
	go func() {
		if err := nsm.registerServer.Serve(nsm.registerSock); err != nil {
			logrus.Fatalf("Failed to start NSM grpc server")
		}
	}()
	endpoints, err := setLocalNSM(model, nsm.serviceRegistry)
	if err != nil {
		logrus.Errorf("failed to set local NSM %+v", err)
		return nil, err
	}

	// Check if the socket of NSM server is operation
	_, conn, err := nsm.serviceRegistry.NSMDApiClient()
	if err != nil {
		nsm.Stop()
		return nil, err
	}
	_ = conn.Close()
	logrus.Infof("NSM gRPC socket: %s is operational", nsm.registerSock.Addr().String())

	// Restore monitors
	nsm.initMonitorServers()

	nsm.remoteServer = remote_server.NewRemoteNetworkServiceServer(nsm.manager, nsm.remoteConnectionMonitor)
	nsm.manager.SetRemoteServer(nsm.remoteServer)

	// Restore existing clients in case of NSMd restart.
	nsm.restore(endpoints)

	return nsm, nil
}

func (nsm *nsmServer) initMonitorServers() {
	nsm.xconManager = services.NewClientConnectionManager(nsm.model, nsm.manager, nsm.serviceRegistry)
	// Start CrossConnect monitor server
	nsm.crossConnectMonitor = monitor_crossconnect.NewMonitorServer()
	// Start Connection monitor server
	nsm.remoteConnectionMonitor = remote.NewMonitorServer(nsm.xconManager)
}

func (nsm *nsmServer) StartDataplaneRegistratorServer() error {
	var err error
	nsm.regServer, err = StartDataplaneRegistrarServer(nsm.model)
	return err
}

func setLocalNSM(model model.Model, serviceRegistry serviceregistry.ServiceRegistry) (*registry.NetworkServiceEndpointList, error) {
	client, err := serviceRegistry.NsmRegistryClient()
	if err != nil {
		err = fmt.Errorf("Failed to get RegistryClient: %s", err)
		return nil, err
	}

	nsm, err := client.RegisterNSM(context.Background(), &registry.NetworkServiceManager{
		Url: serviceRegistry.GetPublicAPI(),
	})
	if err != nil {
		err = fmt.Errorf("Failed to get my own NetworkServiceManager: %s", err)
		return nil, err
	}

	endpoints, err := client.GetEndpoints(context.Background(), &empty.Empty{})
	if err != nil {
		err = fmt.Errorf("Failed to get list of own Endpoints: %s", err)
		return nil, err
	}

	logrus.Infof("Setting local NSM %v", nsm)
	model.SetNsm(nsm)

	return endpoints, nil
}

// StartAPIServerAt starts GRPC API server at sock
func (nsm *nsmServer) StartAPIServerAt(ctx context.Context, sock net.Listener, probes probes.Probes) {
	grpcServer := tools.NewServer()

	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, nsm.crossConnectMonitor)
	connection.RegisterMonitorConnectionServer(grpcServer, nsm.remoteConnectionMonitor)
	probes.Append(health.NewGrpcHealth(grpcServer, sock.Addr(), time.Minute))

	// Register Remote NetworkServiceManager
	remote_networkservice.RegisterNetworkServiceServer(grpcServer, nsm.remoteServer)

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Fatalf("Failed to start NSM API server: %+v", err)
		}
	}()
	logrus.Infof("NSM gRPC API Server: %s is operational", sock.Addr().String())
}
