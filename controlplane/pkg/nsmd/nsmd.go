package nsmd

import (
	"fmt"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"github.com/ligato/networkservicemesh/controlplane/pkg/remote/network_service_server"
	"github.com/ligato/networkservicemesh/controlplane/pkg/serviceregistry"

	"sync"
	"time"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/nsmdapi"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"

	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type nsmServer struct {
	sync.Mutex
	id              int
	workspaces      map[string]*Workspace
	model           model.Model
	serviceRegistry serviceregistry.ServiceRegistry
}

func RequestWorkspace(serviceRegistry serviceregistry.ServiceRegistry) (*nsmdapi.ClientConnectionReply, error) {
	client, con, err := serviceRegistry.NSMDApiClient()
	if err != nil {
		logrus.Fatalf("Failed to connect to NSMD: %+v...", err)
	}
	defer con.Close()

	reply, err := client.RequestClientConnection(context.Background(), &nsmdapi.ClientConnectionRequest{})
	if err != nil {
		return nil, err
	}
	logrus.Infof("nsmd allocated workspace %+v for client operations...", reply)
	return reply, nil
}

func (nsm *nsmServer) RequestClientConnection(context context.Context, request *nsmdapi.ClientConnectionRequest) (*nsmdapi.ClientConnectionReply, error) {
	logrus.Infof("Requested client connection to nsmd : %+v", request)
	nsm.Lock()
	nsm.id++
	id := nsm.id
	nsm.Unlock()

	logrus.Infof("Creating new workspace for: %+v", request)
	workspace, err := NewWorkSpace(nsm.model, nsm.serviceRegistry, fmt.Sprintf("nsm-%d", id))
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
		HostBasedir:     hostBaseDir,
		ClientBaseDir:   clientBaseDir,
		NsmServerSocket: NsmServerSocket,
		NsmClientSocket: NsmClientSocket,
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

func waitForDataplaneAvailable(model model.Model) {
	logrus.Info("Waiting for dataplane available...")
	for ; true; <-time.After(100 * time.Millisecond) {
		if dp, _ := model.SelectDataplane(); dp != nil {
			break
		}
	}
}

func StartNSMServer(model model.Model, serviceRegistry serviceregistry.ServiceRegistry, apiRegistry serviceregistry.ApiRegistry) error {
	if err := tools.SocketCleanup(ServerSock); err != nil {
		return err
	}
	waitForDataplaneAvailable(model)

	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	nsm := nsmServer{
		workspaces:      make(map[string]*Workspace),
		model:           model,
		serviceRegistry: serviceRegistry,
	}
	nsmdapi.RegisterNSMDServer(grpcServer, &nsm)

	sock, err := apiRegistry.NewNSMServerListener()
	if err != nil {
		logrus.Errorf("failed to start device plugin grpc server %+v", err)
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

func StartAPIServer(model model.Model, apiRegistry serviceregistry.ApiRegistry, serviceRegistry serviceregistry.ServiceRegistry) error {
	sock, err := apiRegistry.NewPublicListener()
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)

	// Start Cross connect monitor and server
	startCrossConnectMonitor(grpcServer, model)

	// Register Remote NetworkServiceManager
	network_service_server.NewRemoteNetworkServiceServer(model, serviceRegistry, grpcServer)

	// TODO: Add more public API services here.

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Errorf("failed to start gRPC NSMD API server %+v", err)
		}
	}()
	logrus.Infof("NSM gRPC API Server: %s is operational", sock.Addr().String())

	return nil
}

func startCrossConnectMonitor(grpcServer *grpc.Server, model model.Model) {
	monitor := monitor_crossconnect_server.NewMonitorCrossConnectServer()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitor)
	monitorClient := NewMonitorCrossConnectClient(monitor)
	monitorClient.Register(model)
}
