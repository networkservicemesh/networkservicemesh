package nsmd

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote_connection_monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"os"
	"strings"
	"sync"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsmdapi"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote/network_service_server"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type nsmServer struct {
	sync.Mutex
	workspaces      map[string]*Workspace
	model           model.Model
	serviceRegistry serviceregistry.ServiceRegistry
	manager         nsm.NetworkServiceManager
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

	workspace, err := NewWorkSpace(nsm.model, nsm.manager, nsm.serviceRegistry, request.Workspace)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logrus.Infof("New workspace created: %+v", workspace)

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
	workspace.Close()
	nsm.Lock()
	delete(nsm.workspaces, socket)
	nsm.Unlock()

	return &nsmdapi.DeleteConnectionReply{}, nil
}

func (nsm *nsmServer) EnumConnection(context context.Context, request *nsmdapi.EnumConnectionRequest) (*nsmdapi.EnumConnectionReply, error) {
	nsm.Lock()
	defer nsm.Unlock()
	workspaces := make([]string, len(nsm.workspaces), len(nsm.workspaces))
	for w := range nsm.workspaces {
		workspaces = append(workspaces, w)
	}
	return &nsmdapi.EnumConnectionReply{Workspace: workspaces}, nil
}

func StartNSMServer(model model.Model, manager nsm.NetworkServiceManager, serviceRegistry serviceregistry.ServiceRegistry, apiRegistry serviceregistry.ApiRegistry) error {
	if err := tools.SocketCleanup(ServerSock); err != nil {
		return err
	}
	serviceRegistry.WaitForDataplaneAvailable(model)

	tracer := opentracing.GlobalTracer()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	nsm := nsmServer{
		workspaces:      make(map[string]*Workspace),
		model:           model,
		serviceRegistry: serviceRegistry,
		manager:         manager,
	}
	nsmdapi.RegisterNSMDServer(grpcServer, &nsm)

	sock, err := apiRegistry.NewNSMServerListener()
	if err != nil {
		logrus.Errorf("failed to start device plugin grpc server %+v", err)
		return err
	}
	err = setLocalNSM(model, serviceRegistry)
	if err != nil {
		logrus.Errorf("failed to set local NSM %+v", err)
		return err
	}
	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Error("failed to start device plugin grpc server")
		}
	}()

	// Check if the socket of NSM server is operation
	_, conn, err := serviceRegistry.NSMDApiClient()
	if err != nil {
		return err
	}
	conn.Close()
	logrus.Infof("NSM gRPC socket: %s is operational", sock.Addr().String())

	return nil
}

func setLocalNSM(model model.Model, serviceRegistry serviceregistry.ServiceRegistry) error {
	client, err := serviceRegistry.RegistryClient()
	if err != nil {
		err = fmt.Errorf("Failed to get RegistryClient: %s", err)
		return err
	}
	nsm, err := client.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkServiceManager: &registry.NetworkServiceManager{
			Url: serviceRegistry.GetPublicAPI(),
		},
	})
	if err != nil {
		err = fmt.Errorf("Failed to get my own NetworkServiceManager: %s", err)
		return err
	}
	logrus.Infof("Setting local NSM %v", nsm.GetNetworkServiceManager())
	model.SetNsm(nsm.GetNetworkServiceManager())
	return nil
}

func StartAPIServer(model model.Model, manager nsm.NetworkServiceManager, apiRegistry serviceregistry.ApiRegistry, serviceRegistry serviceregistry.ServiceRegistry) error {
	sock, err := apiRegistry.NewPublicListener()
	if err != nil {
		return err
	}
	xconManager := services.NewClientConnectionManager(model, manager, serviceRegistry)
	tracer := opentracing.GlobalTracer()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	// Start CrossConnect monitor server
	//monitorCrossConnectServer := monitor_crossconnect_server.NewMonitorCrossConnectServer()
	monitorCrossConnectServer := crossconnect_monitor.NewCrossConnectMonitor()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitorCrossConnectServer)

	// Start Connection monitor server
	//monitorConnectionServer := monitor_connection_server.NewMonitorConnectionServer()
	monitorConnectionServer := remote_connection_monitor.NewRemoteConnectionMonitor()
	connection.RegisterMonitorConnectionServer(grpcServer, monitorConnectionServer)

	// Register CrossConnect monitorCrossConnectServer client as ModelListener
	monitorCrossConnectClient := NewMonitorCrossConnectClient(monitorCrossConnectServer, monitorConnectionServer, xconManager)
	model.AddListener(monitorCrossConnectClient)

	// Register Remote NetworkServiceManager
	remoteServer := network_service_server.NewRemoteNetworkServiceServer(model, manager, serviceRegistry, monitorConnectionServer)
	networkservice.RegisterNetworkServiceServer(grpcServer, remoteServer)

	// TODO: Add more public API services here.

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Errorf("failed to start gRPC NSMD API server %+v", err)
		}
	}()
	logrus.Infof("NSM gRPC API Server: %s is operational", sock.Addr().String())

	return nil
}

func GetExcludedPrefixes() []string {
	//TODO: Add a better way to pass this value to NSMD
	excluded_prefixes := []string{}
	exclude_prefixes_env, ok := os.LookupEnv(ExcludedPrefixesEnv)
	if ok {
		for _, s := range strings.Split(exclude_prefixes_env, ",") {
			excluded_prefixes = append(excluded_prefixes, strings.TrimSpace(s))
		}
	}
	return excluded_prefixes
}
