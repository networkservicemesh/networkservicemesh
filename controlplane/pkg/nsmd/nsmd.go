package nsmd

import (
	"fmt"
	"net"
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmdapi"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	ServerSock   = "/var/lib/networkservicemesh/nsm.io.sock"
	clientSocket = "/var/lib/networkservicemesh/nsm-%d.io.sock"
)

type nsmServer struct {
	mux     sync.Mutex
	id      int
	clients map[string]chan bool
	model   model.Model
}

func (nsm *nsmServer) RequestClientConnection(context context.Context, request *nsmdapi.ClientConnectionRequest) (*nsmdapi.ClientConnectionReply, error) {
	logrus.Infof("Requested client connection to nsmd")
	nsm.mux.Lock()
	nsm.id++
	id := nsm.id
	nsm.mux.Unlock()

	socket := fmt.Sprintf(clientSocket, id)
	channel := make(chan bool)
	nsm.clients[socket] = channel
	startClientServer(nsm.model, socket, channel)

	reply := &nsmdapi.ClientConnectionReply{
		Accepted:       true,
		SocketLocation: socket,
	}
	return reply, nil
}

func (nsm *nsmServer) DeleteClientConnection(context context.Context, request *nsmdapi.DeleteConnectionRequest) (*nsmdapi.DeleteConnectionReply, error) {
	socket := request.SocketLocation
	logrus.Infof("Delete connection for socket %s", socket)
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
