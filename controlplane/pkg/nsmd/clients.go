package nsmd

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nseconnect"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
)

type nsmClientServer struct {
	model           model.Model
	socketPath      string
	nsmPodIPAddress string
}

// RequestConnection accepts connection from NSM client and attempts to analyze requested info, call for Dataplane programming and
// return to NSM client result.
func (n *nsmClientServer) RequestConnection(ctx context.Context, cr *nsmconnect.ConnectionRequest) (*nsmconnect.ConnectionReply, error) {
	logrus.Infof("received connection request id: %s, requesting network service: %s for linux namespace: %s",
		cr.RequestId, cr.NetworkServiceName, cr.LinuxNamespace)

	// Need to check if for requested network service, there are advertised Endpoints
	endpoints := n.model.GetNetworkServiceEndpoints(cr.NetworkServiceName)
	endpoints = model.FilterEndpointsByHost(endpoints, n.nsmPodIPAddress)
	endpoints = model.FindEndpointsForMechanism(endpoints, cr.LocalMechanisms)

	if len(endpoints) == 0 {
		return &nsmconnect.ConnectionReply{
			Accepted:       false,
			AdmissionError: fmt.Sprintf("No endpoints registered for Network Service %s", cr.NetworkServiceName),
		}, status.Error(codes.Aborted, "No endpoints registered for Network Service")
	}

	// At this point endpoints contains slice of endpoints matching requested network service and matching client's requested
	// interface type. Until more sofisticated algorithm is proposed, selecting a random entry from the slice.
	src := rand.NewSource(time.Now().Unix())
	rnd := rand.New(src)
	selectedEndpoint := endpoints[rnd.Intn(len(endpoints))]
	logrus.Infof("Endpoint %s selected for network service %s", selectedEndpoint.NseProviderName,
		cr.NetworkServiceName)

	nseConn, err := tools.SocketOperationCheck(selectedEndpoint.SocketLocation)
	if err != nil {
		return &nsmconnect.ConnectionReply{
			Accepted:       false,
			AdmissionError: fmt.Sprintf("Failer to connect endpoint on socket: %s", selectedEndpoint.SocketLocation),
		}, status.Error(codes.Aborted, "Failed to connect endpoint")
	}
	defer nseConn.Close()
	nseClient := nseconnect.NewEndpointConnectionClient(nseConn)

	nseCtx, nseCancel := context.WithTimeout(context.Background(), nseConnectionTimeout)
	defer nseCancel()
	nseRepl, err := nseClient.RequestEndpointConnection(nseCtx, &nseconnect.EndpointConnectionRequest{
		RequestId: cr.RequestId,
	})
	if err != nil {
		return nil, err
	}
	logrus.Infof("successfuly received information from NSE: %v", nseRepl)

	dataplane, err := n.model.SelectDataplane()
	if err != nil {
		return &nsmconnect.ConnectionReply{
			Accepted:       false,
			AdmissionError: fmt.Sprintf("No dataplane available: %v", err),
		}, status.Error(codes.Aborted, "No dataplane")
	}

	logrus.Infof("Programming dataplane: %v...", dataplane)

	return &nsmconnect.ConnectionReply{
		Accepted:             true,
		ConnectionParameters: &nsmconnect.ConnectionParameters{},
	}, nil
}

// Client server starts for each client during Kubelet's Allocate call
func startClientServer(model model.Model, socket string, stopChannel chan bool) {
	client := &nsmClientServer{
		socketPath: socket,
		model:      model,
	}

	if err := tools.SocketCleanup(socket); err != nil {
		return
	}

	unix.Umask(socketMask)
	sock, err := newCustomListener(socket)
	if err != nil {
		logrus.Errorf("failure to listen on socket %s with error: %+v", client.socketPath, err)
		return
	}

	grpcServer := grpc.NewServer()
	// Plugging NSM client Connection methods
	nsmconnect.RegisterClientConnectionServer(grpcServer, client)
	logrus.Infof("Starting Client gRPC server listening on socket: %s", socket)
	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Fatalf("unable to start client grpc server %s, err: %+v", socket, err)
		}
	}()

	conn, err := tools.SocketOperationCheck(socket)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", client.socketPath, err)
		return
	}
	conn.Close()
	logrus.Infof("Client Server socket: %s is operational", socket)

	// TODO: proper shutdown
	go func() {
		select {
		case <-stopChannel:
			logrus.Infof("Server for socket %s received shutdown request", client.socketPath)
			grpcServer.GracefulStop()
		}
		stopChannel <- true
	}()
}
