package nsmd

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	remoteMonitor "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/pkg/probes"
	"github.com/networkservicemesh/networkservicemesh/pkg/probes/health"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsmdapi"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nseregistry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/sdk/monitor/crossconnect"
)

const (
	NsmdDeleteLocalRegistry = "NSMD_LOCAL_REGISTRY_DELETE"
	ForwarderTimeout        = 1 * time.Hour
	NSEAliveTimeout         = 1 * time.Second
)

type NSMServer interface {
	Stop()
	StartForwarderRegistratorServer(ctx context.Context) error
	StartAPIServerAt(ctx context.Context, sock net.Listener, probes probes.Probes)

	XconManager() *services.ClientConnectionManager
	Manager() nsm.NetworkServiceManager

	nsm.MonitorManager
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
	regServer        *ForwarderRegistrarServer

	xconManager             *services.ClientConnectionManager
	crossConnectMonitor     monitor_crossconnect.MonitorServer
	remoteConnectionMonitor remoteMonitor.MonitorServer
	remoteServer            unified.NetworkServiceServer
}

func (nsm *nsmServer) XconManager() *services.ClientConnectionManager {
	return nsm.xconManager
}

func (nsm *nsmServer) Manager() nsm.NetworkServiceManager {
	return nsm.manager
}

func (nsm *nsmServer) LocalConnectionMonitor(workspace string) connectionmonitor.MonitorServer {
	nsm.Lock()
	defer nsm.Unlock()
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

// RequestWorkspace - request a workspace
func RequestWorkspace(ctx context.Context, serviceRegistry serviceregistry.ServiceRegistry, id string) (*nsmdapi.ClientConnectionReply, error) {
	span := spanhelper.FromContext(ctx, "RequestWorkspace")
	defer span.Finish()
	client, con, err := serviceRegistry.NSMDApiClient(span.Context())
	if err != nil {
		span.Logger().Fatalf("Failed to connect to NSMD: %+v...", err)
	}
	defer con.Close()

	reply, err := client.RequestClientConnection(ctx, &nsmdapi.ClientConnectionRequest{Workspace: id})
	span.LogError(err)
	if err != nil {
		return nil, err
	}
	span.LogObject("response", reply)
	span.Logger().Infof("nsmd allocated workspace %+v for client operations...", reply)
	return reply, nil
}

func (nsm *nsmServer) RequestClientConnection(ctx context.Context, request *nsmdapi.ClientConnectionRequest) (*nsmdapi.ClientConnectionReply, error) {
	span := spanhelper.FromContext(ctx, "RequestClientConnection")
	defer span.Finish()

	span.Logger().Infof("Requested client connection to nsmd : %+v", request)

	nsm.Lock()
	workspace, ok := nsm.workspaces[request.Workspace]
	nsm.Unlock()
	if ok {
		span.Logger().Infof("workspace %s already exists. reusing it.", request.Workspace)
		reply := &nsmdapi.ClientConnectionReply{
			Workspace:       workspace.Name(),
			HostBasedir:     workspace.locationProvider.HostBaseDir(),
			ClientBaseDir:   workspace.locationProvider.ClientBaseDir(),
			NsmServerSocket: workspace.locationProvider.NsmServerSocket(),
			NsmClientSocket: workspace.locationProvider.NsmClientSocket(),
		}
		span.LogObject("ClientConnectionReply", reply)
		return reply, nil
	}

	workspace, err := NewWorkSpace(span.Context(), nsm, request.Workspace, false)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.Logger().Infof("New workspace created: %+v", workspace)

	err = nsm.localRegistry.AppendClientRequest(workspace.Name())
	if err != nil {
		span.LogError(errors.Wrap(err, "failed to store Client information into local registry: %v"))
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
	span.LogObject("ClientConnectionReply", reply)
	return reply, nil
}

func (nsm *nsmServer) DeleteClientConnection(context context.Context, request *nsmdapi.DeleteConnectionRequest) (*nsmdapi.DeleteConnectionReply, error) {
	socket := request.Workspace
	logrus.Infof("Delete connection for workspace %s", socket)

	workspace, ok := nsm.workspaces[socket]
	if !ok {
		err := errors.Errorf("no connection exists for workspace %s", socket)
		return &nsmdapi.DeleteConnectionReply{}, err
	}

	err := nsm.localRegistry.DeleteClient(workspace.Name())
	if err != nil {
		logrus.Errorf("failed to delete Client information into local registry: %v", err)
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

func (nsm *nsmServer) restore(ctx context.Context, registeredEndpointsList *registry.NetworkServiceEndpointList) {
	span := spanhelper.FromContext(ctx, "restore")
	defer span.Finish()

	deleteClientRegistry := os.Getenv(NsmdDeleteLocalRegistry) == "true"
	span.LogValue("delete.client.registry", deleteClientRegistry)
	if deleteClientRegistry {
		logrus.Errorf("delete of local nse/client registry... by ENV VAR: %s", NsmdDeleteLocalRegistry)
		nsm.localRegistry.Delete()
	}

	clients, nses, err := nsm.localRegistry.LoadRegistry()
	if err != nil {
		span.LogError(errors.Errorf("error Loading stored NSE registry: %v", err))
		return
	}

	span.LogObject("clients", clients)
	span.LogObject("endpoints", nses)

	registeredNSEs := map[string]string{}
	for _, endpoint := range registeredEndpointsList.GetNetworkServiceEndpoints() {
		registeredNSEs[endpoint.GetName()] = endpoint.GetNetworkServiceName()
	}

	updatedClients := nsm.restoreClients(span.Context(), clients)
	span.LogObject("updated-clients", updatedClients)
	updatedEndpoints, err := nsm.restoreEndpoints(span.Context(), nses, registeredNSEs)
	span.LogObject("updated-endpoints", updatedEndpoints)
	if err != nil {
		span.LogError(errors.Wrap(err, "error restoring endpoints"))
		return
	}

	if len(updatedClients) > 0 || len(updatedEndpoints) > 0 {
		span.Logger().Infof("save local client/nse registry")
		if err := nsm.localRegistry.Save(updatedClients, updatedEndpoints); err != nil {
			span.LogError(errors.Wrap(err, "store updated NSE local registry..."))
		}
	}

	span.Logger().Info("NSMD: Restore of NSE/Clients Complete...")
}

func (nsm *nsmServer) restoreClients(ctx context.Context, clients []string) []string {
	nsm.Lock()
	defer nsm.Unlock()

	span := spanhelper.FromContext(ctx, "restoreClients")
	defer span.Finish()

	span.Logger().Infof("NSMServer: Creating workspaces for existing clients...")

	updatedClients := make([]string, 0, len(clients))
	for _, client := range clients {
		if client == "" {
			span.Logger().Info("client is empty, skip")
			continue
		}
		workspace, err := NewWorkSpace(span.Context(), nsm, client, true)
		if err != nil {
			span.LogError(errors.Wrapf(err, "error NSMServer: Failed to create workspace %s. Ignoring", client))
			continue
		}
		nsm.workspaces[workspace.Name()] = workspace
		updatedClients = append(updatedClients, client)
		span.Logger().Infof("Workspace for client %v created", client)
	}

	return updatedClients
}

func (nsm *nsmServer) restoreEndpoints(ctx context.Context, nses map[string]nseregistry.NSEEntry, registeredNSEs map[string]string) (map[string]nseregistry.NSEEntry, error) {
	span := spanhelper.FromContext(ctx, "restore-endpoints")
	defer span.Finish()
	discoveryClient, err := nsm.serviceRegistry.DiscoveryClient(span.Context())
	if err != nil {
		span.LogError(errors.Wrap(err, "failed to get DiscoveryClient: %v"))
		return nil, err
	}

	registryClient, err := nsm.serviceRegistry.NseRegistryClient(span.Context())
	if err != nil {
		span.LogError(errors.Wrap(err, "failed to get RegistryClient: %v"))
		return nil, err
	}

	networkServices := map[string]bool{}
	updatedNSEs := map[string]nseregistry.NSEEntry{}
	for name, nse := range nses {
		ws, ok := nsm.workspaces[nse.Workspace]
		if !ok {
			continue
		}

		nseSpan := spanhelper.FromContext(span.Context(), fmt.Sprintf("check-nse-%v", name))
		nseSpan.LogObject("workspace", ws)

		nseSpan.Logger().Infof("Checking NSE %s is alive at %v...", name, ws.NsmClientSocket())
		if !ws.isConnectionAlive(nseSpan.Context(), NSEAliveTimeout) {
			span.Logger().Errorf("unable to connect to local nse %v. Skipping", nse.NseReg)
			if err := nsm.deleteEndpointWithClient(span.Context(), name, registryClient); err != nil {
				span.Logger().Errorf("remove NSE: NSE %v", err)
			}
			nseSpan.Finish()
			continue
		}

		span.Logger().Infof("NSE %s is alive at %v...", name, ws.NsmClientSocket())
		newName, newNSE, err := nsm.restoreEndpoint(span.Context(), discoveryClient, registryClient, name, nse, ws, registeredNSEs, networkServices)
		if err != nil {
			span.LogError(errors.Wrap(err, "failed to register NSE: %v"))
			nseSpan.Finish()
			continue
		}

		networkServices[newNSE.NseReg.GetNetworkService().GetName()] = true
		updatedNSEs[newName] = newNSE
		nseSpan.Finish()
	}

	// We need to unregister NSE's without NSM registration.
	for name := range registeredNSEs {
		if _, ok := updatedNSEs[name]; !ok {
			if _, err := registryClient.RemoveNSE(span.Context(), &registry.RemoveNSERequest{
				NetworkServiceEndpointName: name,
			}); err != nil {
				span.LogError(errors.Wrap(err, "remove NSE: NSE %v"))
			}
		}
	}

	return updatedNSEs, nil
}

func (nsm *nsmServer) restoreEndpoint(ctx context.Context,
	discoveryClient registry.NetworkServiceDiscoveryClient,
	registryClient registry.NetworkServiceRegistryClient,
	name string,
	nse nseregistry.NSEEntry,
	ws *Workspace,
	registeredNSEs map[string]string,
	networkServices map[string]bool) (string, nseregistry.NSEEntry, error) {
	span := spanhelper.FromContext(ctx, "restoreEndpoint")
	defer span.Finish()

	if networkService, ok := registeredNSEs[name]; ok {
		if networkServices[networkService] {
			nsm.restoreRegisteredEndpoint(span.Context(), nse, ws)
			return name, nse, nil
		}

		if _, err := discoveryClient.FindNetworkService(span.Context(), &registry.FindNetworkServiceRequest{
			NetworkServiceName: networkService,
		}); err == nil {
			nsm.restoreRegisteredEndpoint(span.Context(), nse, ws)
			return name, nse, nil
		}

		if err := nsm.deleteEndpointWithClient(span.Context(), name, registryClient); err != nil {
			return "", nseregistry.NSEEntry{}, err
		}
	}

	return nsm.restoreNotRegisteredEndpoint(span.Context(), registryClient, nse, ws)
}

func (nsm *nsmServer) restoreRegisteredEndpoint(ctx context.Context, nse nseregistry.NSEEntry, ws *Workspace) {
	nse.NseReg.NetworkServiceManager = nsm.model.GetNsm()
	nse.NseReg.NetworkServiceEndpoint.NetworkServiceManagerName = nse.NseReg.GetNetworkServiceManager().GetName()

	nsm.model.AddEndpoint(ctx, &model.Endpoint{
		Endpoint:       nse.NseReg,
		Workspace:      nse.Workspace,
		SocketLocation: ws.NsmClientSocket(),
	})
}

func (nsm *nsmServer) restoreNotRegisteredEndpoint(ctx context.Context,
	registryClient registry.NetworkServiceRegistryClient,
	nse nseregistry.NSEEntry,
	ws *Workspace) (string, nseregistry.NSEEntry, error) {
	name := nse.NseReg.GetNetworkServiceEndpoint().GetName()
	span := spanhelper.FromContext(ctx, fmt.Sprintf("restoreNotRegisteredEndpoint-%v", name))
	span.LogObject("name", name)
	defer span.Finish()

	reg, err := ws.registryServer.RegisterNSEWithClient(span.Context(), nse.NseReg, registryClient)
	span.LogObject("reg-response", reg)
	if err != nil {
		span.LogError(err)

		nse.NseReg.NetworkServiceEndpoint.Name = ""
		reg, err = ws.registryServer.RegisterNSEWithClient(span.Context(), nse.NseReg, registryClient)
		if err != nil {
			span.LogError(err)
			return "", nseregistry.NSEEntry{}, err
		}
		span.LogObject("new-name", reg)

		nsm.manager.NotifyRenamedEndpoint(name, reg.GetNetworkServiceEndpoint().GetName())
	}

	return reg.GetNetworkServiceEndpoint().GetName(), nseregistry.NSEEntry{
		Workspace: ws.Name(),
		NseReg:    reg,
	}, nil
}

func (nsm *nsmServer) deleteEndpointWithClient(ctx context.Context, name string, client registry.NetworkServiceRegistryClient) error {
	if _, err := client.RemoveNSE(ctx, &registry.RemoveNSERequest{
		NetworkServiceEndpointName: name,
	}); err != nil {
		return err
	}

	nsm.model.DeleteEndpoint(ctx, name)

	return nil
}

// DeleteEndpointWithBrokenConnection deletes endpoint from the model and k8s-registry
func (nsm *nsmServer) DeleteEndpointWithBrokenConnection(ctx context.Context, endpoint *model.Endpoint) error {
	span := spanhelper.FromContext(ctx, "DeleteEndpointWithBrokenConnection")
	defer span.Finish()

	client, err := nsm.serviceRegistry.NseRegistryClient(span.Context())
	if err != nil {
		return err
	}

	return nsm.deleteEndpointWithClient(span.Context(), endpoint.EndpointName(), client)
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
	span := spanhelper.FromContext(ctx, "nsm-server-start")
	defer span.Finish()

	var err error
	if err = tools.SocketCleanup(ServerSock); err != nil {
		span.LogError(err)
		return nil, err
	}

	locationProvider := manager.ServiceRegistry().NewWorkspaceProvider()

	nsm := createNsmServer(model, manager, locationProvider)

	span.Logger().Infof("Starting NSM server")

	nsm.registerServer = tools.NewServer(span.Context())
	nsmdapi.RegisterNSMDServer(nsm.registerServer, nsm)

	nsm.registerSock, err = apiRegistry.NewNSMServerListener()
	if err != nil {
		span.LogError(errors.Wrap(err, "failed to start device plugin grpc server"))
		nsm.Stop()
		return nil, err
	}
	go func() {
		if err := nsm.registerServer.Serve(nsm.registerSock); err != nil {
			span.Logger().Fatalf("failed to start NSM grpc server: %v", err)
		}
	}()
	endpoints, err := setLocalNSM(span.Context(), model, nsm.serviceRegistry)
	span.LogObject("registered-endpoints", endpoints)
	if err != nil {
		span.LogError(errors.Wrap(err, "failed to set local NSM"))
		return nil, err
	}

	// Check if the socket of NSM server is operation
	_, conn, err := nsm.serviceRegistry.NSMDApiClient(span.Context())
	if err != nil {
		nsm.Stop()
		return nil, err
	}
	_ = conn.Close()
	span.Logger().Infof("NSM gRPC socket: %s is operational", nsm.registerSock.Addr().String())

	// Restore monitors
	span.Logger().Infof("create monitor servers")
	nsm.initMonitorServers()

	nsm.remoteServer = remote.NewRemoteNetworkServiceServer(nsm.manager, nsm.remoteConnectionMonitor)
	nsm.manager.SetRemoteServer(nsm.remoteServer)

	// Restore existing clients in case of NSMd restart.
	nsm.restore(span.Context(), endpoints)

	return nsm, nil
}

func createNsmServer(model model.Model, manager nsm.NetworkServiceManager, locationProvider serviceregistry.WorkspaceLocationProvider) *nsmServer {
	nsm := &nsmServer{
		workspaces:       make(map[string]*Workspace),
		model:            model,
		serviceRegistry:  manager.ServiceRegistry(),
		manager:          manager,
		locationProvider: locationProvider,
		localRegistry:    nseregistry.NewNSERegistry(locationProvider.NsmNSERegistryFile()),
	}
	return nsm
}

func (nsm *nsmServer) initMonitorServers() {
	nsm.xconManager = services.NewClientConnectionManager(nsm.model, nsm.manager, nsm.serviceRegistry)
	// Start CrossConnect monitor server
	nsm.crossConnectMonitor = monitor_crossconnect.NewMonitorServer()
	// Start Connection monitor server
	nsm.remoteConnectionMonitor = remoteMonitor.NewMonitorServer(nsm.xconManager)
}

func (nsm *nsmServer) StartForwarderRegistratorServer(ctx context.Context) error {
	var err error
	nsm.regServer, err = StartForwarderRegistrarServer(ctx, nsm.model)
	return err
}

func setLocalNSM(ctx context.Context, model model.Model, serviceRegistry serviceregistry.ServiceRegistry) (*registry.NetworkServiceEndpointList, error) {
	span := spanhelper.FromContext(ctx, "set-local-nsm")
	defer span.Finish()
	client, err := serviceRegistry.NsmRegistryClient(span.Context())
	if err != nil {
		err = errors.Wrap(err, "failed to get RegistryClient")
		return nil, err
	}
	span.LogValue("url", serviceRegistry.GetPublicAPI())

	nsm, err := client.RegisterNSM(span.Context(), &registry.NetworkServiceManager{
		Url: serviceRegistry.GetPublicAPI(),
	})
	if err != nil {
		err = errors.Wrap(err, "failed to get my own NetworkServiceManager")
		return nil, err
	}

	endpoints, err := client.GetEndpoints(span.Context(), &empty.Empty{})
	if err != nil {
		err = errors.Wrap(err, "failed to get list of own Endpoints")
		return nil, err
	}

	logrus.Infof("Setting local NSM %v", nsm)
	model.SetNsm(nsm)

	return endpoints, nil
}

// StartAPIServerAt starts GRPC API server at sock
func (nsm *nsmServer) StartAPIServerAt(ctx context.Context, sock net.Listener, probes probes.Probes) {
	span := spanhelper.FromContext(ctx, "start-public-api-server")
	defer span.Finish()

	grpcServer := tools.NewServer(span.Context())

	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, nsm.crossConnectMonitor)
	connection.RegisterMonitorConnectionServer(grpcServer, nsm.remoteConnectionMonitor)
	probes.Append(health.NewGrpcHealth(grpcServer, sock.Addr(), time.Minute))

	// Register Remote NetworkServiceManager
	unified.RegisterNetworkServiceServer(grpcServer, nsm.remoteServer)

	// TODO: Add more public API services here.
	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			span.Logger().Fatalf("Failed to start NSM API server: %+v", err)
		}
	}()
	span.Logger().Infof("NSM gRPC API Server: %s is operational", sock.Addr().String())
}
