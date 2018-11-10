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
	mux     sync.Mutex
	id      int
	clients map[string]chan bool
	model   model.Model
}

func RequestWorkspace() (string, error) {
	logrus.Infof("Connecting to nsmd on socket: %s...", ServerSock)
	if _, err := os.Stat(ServerSock); err != nil {
		return "", err
	}

	conn, err := tools.SocketOperationCheck(ServerSock)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	logrus.Info("Requesting nsmd for client connection...")
	client := nsmdapi.NewNSMDClient(conn)
	reply, err := client.RequestClientConnection(context.Background(), &nsmdapi.ClientConnectionRequest{})
	if err != nil {
		return "", err
	}
	logrus.Infof("nsmd allocated workspace %s for client operations...", reply.Workspace)
	return reply.Workspace, nil
}

func (nsm *nsmServer) RequestClientConnection(context context.Context, request *nsmdapi.ClientConnectionRequest) (*nsmdapi.ClientConnectionReply, error) {
	logrus.Infof("Requested client connection to nsmd")
	nsm.mux.Lock()
	nsm.id++
	id := nsm.id
	nsm.mux.Unlock()

	workspace := fmt.Sprintf("/var/lib/networkservicemesh/nsm-%d", id)
	if err := os.MkdirAll(workspace, folderMask); err != nil {
		logrus.Errorf("can't create folder: %s, error: %v", workspace, err)
		return &nsmdapi.ClientConnectionReply{
			Accepted:  false,
			Workspace: "",
		}, nil
	}

	channel := make(chan bool)
	nsm.clients[workspace] = channel
	startNetworkServiceServer(nsm.model, workspace, channel)

	reply := &nsmdapi.ClientConnectionReply{
		Accepted:  true,
		Workspace: workspace,
	}
	return reply, nil
}

func (nsm *nsmServer) DeleteClientConnection(context context.Context, request *nsmdapi.DeleteConnectionRequest) (*nsmdapi.DeleteConnectionReply, error) {
	socket := request.Workspace
	logrus.Infof("Delete connection for workspace %s", socket)
	channel := nsm.clients[socket]
	channel <- true
	delete(nsm.clients, socket)

	reply := &nsmdapi.DeleteConnectionReply{
		Success: true,
	}
	return reply, nil
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
		clients: make(map[string]chan bool),
		model:   model,
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
