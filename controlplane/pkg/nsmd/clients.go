package nsmd

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
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

// TODO (sbezverk) Current assumption is that NSM client is requesting connection for  NetworkService
// from the same namespace. If it changes, refactor maybe required.
func isInProgress(networkService *clientNetworkService) bool {
	return networkService.isInProgress
}

// type nsmSocket struct {
// 	socketPath  string
// 	stopChannel chan bool
// 	allocated   bool
// }

// clientNetworkService struct represents requested by a NSM client NetworkService and its state, isInProgress true
// indicates that DataPlane programming operation is on going, so no duplicate request for Dataplane processing should occur.
type clientNetworkService struct {
	networkService       *netmesh.NetworkService
	endpoint             *netmesh.NetworkServiceEndpoint
	ConnectionParameters *nsmconnect.ConnectionParameters
	// isInProgress indicates ongoing dataplane programming
	isInProgress bool
}

type nsmClientServer struct {
	model      model.Model
	socketPath string
	// POD UID is used as a unique key in clientConnections map
	// Second key is NetworkService name
	clientConnections map[string]map[string]*clientNetworkService
	sync.RWMutex
	//namespace       string
	nsmPodIPAddress string
}

// getLocalEndpoint returns a slice of nsmapi.NetworkServiceEndpoint with only
// entries matching NSM Pod ip address.
func getLocalEndpoint(endpointList []*netmesh.NetworkServiceEndpoint, nsmPodIPAddress string) []*netmesh.NetworkServiceEndpoint {
	localEndpoints := []*netmesh.NetworkServiceEndpoint{}
	for _, ep := range endpointList {
		if ep.NetworkServiceHost == nsmPodIPAddress {
			localEndpoints = append(localEndpoints, ep)
		}
	}
	return localEndpoints
}

// getEndpointWithInterface returns a slice of slice of nsmapi.NetworkServiceEndpoint with
// only Endpoints offerring correct Interface type. Interface type comes from Client's Connection Request.
func getEndpointWithInterface(endpointList []*netmesh.NetworkServiceEndpoint, reqInterfacesSorted []*common.LocalMechanism) []*netmesh.NetworkServiceEndpoint {
	endpoints := []*netmesh.NetworkServiceEndpoint{}
	found := false
	// Loop over a list of required interfaces, since it is sorted, the loop starts with first choice.
	// if no first choice matches found, loop goes to the second choice, etc., otherwise function
	// returns collected slice of endpoints with matching interface type.
	for _, iReq := range reqInterfacesSorted {
		for _, ep := range endpointList {
			for _, intf := range ep.LocalMechanisms {
				if iReq.Type == intf.Type {
					found = true
					endpoints = append(endpoints, ep)
				}
			}
		}
		if found {
			break
		}
	}
	return endpoints
}

// RequestConnection accepts connection from NSM client and attempts to analyze requested info, call for Dataplane programming and
// return to NSM client result.
func (n *nsmClientServer) RequestConnection(ctx context.Context, cr *nsmconnect.ConnectionRequest) (*nsmconnect.ConnectionReply, error) {
	logrus.Infof("received connection request id: %s, requesting network service: %s for linux namespace: %s",
		cr.RequestId, cr.NetworkServiceName, cr.LinuxNamespace)

	// first check to see if requested NetworkService exists in objectStore
	ns := n.model.GetNetworkService(cr.NetworkServiceName)
	if ns == nil {
		// Unknown NetworkService fail Connection request
		logrus.Errorf("not found network service object: %s", cr.RequestId)
		return &nsmconnect.ConnectionReply{
			Accepted:       false,
			AdmissionError: fmt.Sprintf("requested Network Service %s does not exist", cr.RequestId),
		}, status.Error(codes.NotFound, "requested network service not found")
	}
	logrus.Infof("Requested network service: %s, found network service object", cr.NetworkServiceName)

	// second check to see if requested NetworkService exists in n.clientConnections which means it is not first
	// Connection request
	if _, ok := n.clientConnections[cr.RequestId]; ok {
		// Check if exisiting request for already requested NetworkService
		if _, ok := n.clientConnections[cr.RequestId][cr.NetworkServiceName]; ok {
			// Since it is duplicate request, need to check if it is already inProgress
			if isInProgress(n.clientConnections[cr.RequestId][cr.NetworkServiceName]) {
				// Looks like dataplane programming is taking long time, responding client to wait and retry
				return &nsmconnect.ConnectionReply{
					Accepted:       false,
					AdmissionError: fmt.Sprintf("dataplane for requested Network Service %s is still being programmed, retry", cr.RequestId),
				}, status.Error(codes.AlreadyExists, "dataplane for requested network service is being programmed, retry")
			}
			// Request is not inProgress which means potentially a success can be returned
			// TODO (sbezverk) discuss this logic in case some corner cases might break it.
			return &nsmconnect.ConnectionReply{
				Accepted:             true,
				ConnectionParameters: &nsmconnect.ConnectionParameters{},
			}, nil
		}
	}

	// Need to check if for requested network service, there are advertised Endpoints
	endpointList := n.model.GetNetworkServiceEndpoints(cr.NetworkServiceName)
	if endpointList == nil {
		return &nsmconnect.ConnectionReply{
			Accepted: false,
			AdmissionError: fmt.Sprintf("connection request %s failed to get a list of endpoints for requested Network Service %s",
				cr.RequestId, cr.NetworkServiceName),
		}, status.Error(codes.Aborted, "failed to get a list of endpoints for requested Network Service")
	}
	if len(endpointList) == 0 {
		return &nsmconnect.ConnectionReply{
			Accepted: false,
			AdmissionError: fmt.Sprintf("connection request %s failed no endpoints were found for requested Network Service %s",
				cr.RequestId, cr.NetworkServiceName),
		}, status.Error(codes.NotFound, "failed no endpoints were found for requested Network Service")
	}

	// At this point there is a list of Endpoints providing a network service requested by the client.
	// The following code goes through a selection process, starting from identifying local to NSM
	// Endpoints.
	// TODO (sbezverk) When support for Remote Endpoints get added, then this error message should be removed.
	endpoints := getLocalEndpoint(endpointList, n.nsmPodIPAddress)
	if len(endpoints) == 0 {
		logrus.Errorf("connection request %s failed no local endpoints were found for requested Network Service %s, but remote endpoints are not yet supported",
			cr.RequestId, cr.NetworkServiceName)
		return &nsmconnect.ConnectionReply{
			Accepted: false,
			AdmissionError: fmt.Sprintf("connection request %s failed no local endpoints were found for requested Network Service %s, but remote endpoints are not yet supported",
				cr.RequestId, cr.NetworkServiceName),
		}, status.Error(codes.NotFound, "failed no local endpoints were found for requested Network Service")
	}

	// getEndpointWithInterface returns a slice of slice of nsmapi.NetworkServiceEndpoint with
	// only Endpoints offerring correct Interface type. Interface type comes from Client's Connection Request.
	endpoints = getEndpointWithInterface(endpoints, cr.LocalMechanisms)
	if len(endpoints) == 0 {
		logrus.Errorf("no advertised endpoints for Network Service %s, support required interface", cr.NetworkServiceName)
		return &nsmconnect.ConnectionReply{
			Accepted:       false,
			AdmissionError: fmt.Sprintf("no advertised endpoints for Network Service %s, support required interface", cr.NetworkServiceName),
		}, status.Error(codes.NotFound, "required interface type not found")
	}

	// At this point endpoints contains slice of endpoints matching requested network service and matching client's requested
	// interface type. Until more sofisticated algorithm is proposed, selecting a random entry from the slice.
	src := rand.NewSource(time.Now().Unix())
	rnd := rand.New(src)
	selectedEndpoint := endpoints[rnd.Intn(len(endpoints))]
	logrus.Infof("Endpoint %s selected for network service %s", selectedEndpoint.NseProviderName,
		cr.NetworkServiceName)

	// Add new Connection Request into n.clientConnection, set as inProgress and call DataPlane programming func
	// and wait for complition.
	clientNS := clientNetworkService{
		networkService: &netmesh.NetworkService{
			NetworkServiceName: cr.NetworkServiceName,
		},
		endpoint:             selectedEndpoint,
		isInProgress:         true,
		ConnectionParameters: &nsmconnect.ConnectionParameters{},
	}
	n.Lock()
	n.clientConnections[cr.RequestId] = make(map[string]*clientNetworkService, 0)
	n.clientConnections[cr.RequestId][cr.NetworkServiceName] = &clientNS
	n.Unlock()

	// At this point we have all information to call Connection Request to NSE providing requested NetworkSerice.
	// There are three path where selectedEndpoint points to:
	// 1 - local NSE,
	// 2 - remote NSE (not implmented), local NSM contacts remote NSM and in case of success attempts to build local
	//     end of a tunnel between NSM client pod and NSE providiong requested service.
	// 3 - Network Service Wiring (not implemented).

	// Local NSE case
	if selectedEndpoint.NetworkServiceHost == n.nsmPodIPAddress {
		if err := localNSE(n, cr.RequestId, cr.NetworkServiceName); err != nil {
			logrus.Errorf("nsm: failed to communicate with local NSE over the socket %s with error: %+v", selectedEndpoint.SocketLocation, err)
			cleanConnectionRequest(cr.RequestId, n)
			return &nsmconnect.ConnectionReply{
				Accepted:       false,
				AdmissionError: fmt.Sprintf("failed to communicate with local NSE for requested Network Service %s with error: %+v", cr.NetworkServiceName, err),
			}, status.Error(codes.Aborted, "communication failure with local NSE")
		}
		logrus.Infof("successfully create client connection for request id: %s networkservice: %s",
			cr.RequestId, cr.NetworkServiceName)

		// nsm client requesting connection is one time operation and it does not seem require to keep state
		// after it either succeeded or failed. It seems safe to delete completed Connection Request.
		cleanConnectionRequest(cr.RequestId, n)
		return &nsmconnect.ConnectionReply{
			Accepted:             true,
			ConnectionParameters: &nsmconnect.ConnectionParameters{},
		}, nil
	}
	// Remote NSE case (not implemented)
	logrus.Error("nsm: connection with remote NSE is not implemented, come back later")
	cleanConnectionRequest(cr.RequestId, n)
	return &nsmconnect.ConnectionReply{
		Accepted:       false,
		AdmissionError: fmt.Sprintf("connection with remote NSE is not implemented, come back later"),
	}, status.Error(codes.Aborted, "connection with remote NSE is not implemented, come back later")
}

func cleanConnectionRequest(requestID string, n *nsmClientServer) {
	n.Lock()
	delete(n.clientConnections, requestID)
	n.Unlock()
}

func localNSE(n *nsmClientServer, requestID, networkServiceName string) error {
	client := n.clientConnections[requestID][networkServiceName]
	nseConn, err := tools.SocketOperationCheck(client.endpoint.SocketLocation)
	if err != nil {
		return err
	}
	defer nseConn.Close()
	nseClient := nseconnect.NewEndpointConnectionClient(nseConn)

	nseCtx, nseCancel := context.WithTimeout(context.Background(), nseConnectionTimeout)
	defer nseCancel()
	nseRepl, err := nseClient.RequestEndpointConnection(nseCtx, &nseconnect.EndpointConnectionRequest{
		RequestId: requestID,
	})
	if err != nil {
		return err
	}
	logrus.Infof("successfuly received information from NSE: %s", nseRepl.RequestId)

	// TODO (sbezverk) It must be refactor as soon as possible to call dataplane interface

	logrus.Infof("Call dataplane to interconnect containers")

	return nil
}

// Client server starts for each client during Kubelet's Allocate call
func startClientServer(socket string, stopChannel chan bool) {
	var client nsmClientServer
	client.socketPath = socket

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
	nsmconnect.RegisterClientConnectionServer(grpcServer, &client)
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

	// Wait for shutdown
	select {
	case <-stopChannel:
		logrus.Infof("Server for socket %s received shutdown request", client.socketPath)
	}
	stopChannel <- true
}
