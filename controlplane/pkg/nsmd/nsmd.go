package nsmd

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmdapi"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	ServerSock         = "/var/lib/networkservicemesh/nsm.io.sock"
	DefaultWorkspace   = "/var/lib/networkservicemesh"
	ClientSocket       = "nsm.client.io.sock"
	NsmDevicePluginEnv = "NSM_DEVICE_PLUGIN"
	folderMask         = 0777
)

type nsmServer struct {
	sync.Mutex
	id         int
	workspaces map[string]*Workspace
	model      model.Model
}

func RequestWorkspace() (*nsmdapi.ClientConnectionReply, error) {
	logrus.Infof("Connecting to nsmd on socket: %s...", ServerSock)
	if _, err := os.Stat(ServerSock); err != nil {
		return nil, err
	}

	conn, err := tools.SocketOperationCheck(ServerSock)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	logrus.Info("Requesting nsmd for client connection...")
	client := nsmdapi.NewNSMDClient(conn)
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
	workspace, err := NewWorkSpace(nsm.model, fmt.Sprintf("nsm-%d", id))
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
		err := fmt.Errorf("No connection exists for workspace %s", socket)
		return &nsmdapi.DeleteConnectionReply{}, err
	}
	workspace.Close()
	nsm.Lock()
	delete(nsm.workspaces, socket)
	nsm.Unlock()

	return &nsmdapi.DeleteConnectionReply{}, nil
}

func StartNSMServer(model model.Model) error {
	if err := tools.SocketCleanup(ServerSock); err != nil {
		return err
	}
	sock, err := net.Listen("unix", ServerSock)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	nsm := nsmServer{
		workspaces: make(map[string]*Workspace),
		model:      model,
	}
	nsmdapi.RegisterNSMDServer(grpcServer, &nsm)

	logrus.Infof("Starting NSM gRPC server listening on socket: %s", ServerSock)
	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Error("failed to start device plugin grpc server")
		}
	}()
	// Check if the socket of NSM server is operation
	conn, err := tools.SocketOperationCheck(ServerSock)
	if err != nil {
		return err
	}
	conn.Close()
	logrus.Infof("NSM gRPC socket: %s is operational", ServerSock)

	return nil
}
