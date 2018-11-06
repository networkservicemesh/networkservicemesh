// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"net"
	"os"
	"path"
	"sync"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nseconnect"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	// networkServiceName defines Network Service Name the NSE is serving for
	networkServiceName = "gold-network"
	// EndpointSocketBaseDir defines the location of NSM Endpoints listen socket
	EndpointSocketBaseDir = "/var/lib/networkservicemesh"
	// EndpointSocket defines the name of NSM Endpoints operations socket
	EndpointSocket = "nsm.endpoint.io.sock"
)

type nseConnection struct {
	networkServiceName string
	linuxNamespace     string
}

func (n nseConnection) RequestEndpointConnection(ctx context.Context, req *nseconnect.EndpointConnectionRequest) (*nseconnect.EndpointConnectionReply, error) {

	return &nseconnect.EndpointConnectionReply{
		RequestId:          n.linuxNamespace,
		NetworkServiceName: n.networkServiceName,
		LinuxNamespace:     n.linuxNamespace,
	}, nil
}

func (n nseConnection) SendEndpointConnectionMechanism(ctx context.Context, req *nseconnect.EndpointConnectionMechanism) (*nseconnect.EndpointConnectionMechanismReply, error) {

	return &nseconnect.EndpointConnectionMechanismReply{
		RequestId:      req.RequestId,
		MechanismFound: true,
	}, nil
}

func main() {
	var wg sync.WaitGroup

	// For NSE to program container's dataplane, container's linux namespace must be sent to NSM
	linuxNS, err := tools.GetCurrentNS()
	if err != nil {
		logrus.Fatalf("nse: failed to get a linux namespace with error: %+v, exiting...", err)
		os.Exit(1)
	}
	logrus.Infof("Starting NSE, linux namespace: %s", linuxNS)

	var workspace string
	var perPodDirectory string

	if os.Getenv(nsmd.NsmDevicePluginEnv) != "" {
		workspace = nsmd.DefaultWorkspace
		perPodDirectory = os.Getenv(nsmd.NsmPerPodDirectoryEnv)
	} else {
		workspace, err = nsmd.RequestWorkspace()
		if err != nil {
			logrus.Fatalf("nsc: failed set up client connection, error: %+v, exiting...", err)
			os.Exit(1)
		}
		_, perPodDirectory = path.Split(workspace)
	}

	// NSM socket path will be used to drop NSE socket for NSM's Connection request
	connectionServerSocket := path.Join(workspace, linuxNS+".nse.io.sock")
	if err := tools.SocketCleanup(connectionServerSocket); err != nil {
		logrus.Fatalf("nse: failure to cleanup stale socket %s with error: %+v", connectionServerSocket, err)
	}

	logrus.Infof("nse: listening socket %s", connectionServerSocket)
	connectionServer, err := net.Listen("unix", connectionServerSocket)
	if err != nil {
		logrus.Fatalf("nse: failure to listen on a socket %s with error: %+v", connectionServerSocket, err)
	}
	grpcServer := grpc.NewServer()

	// Registering NSE API, it will listen for Connection requests from NSM and return information
	// needed for NSE's dataplane programming.
	nseConn := nseConnection{
		networkServiceName: networkServiceName,
		linuxNamespace:     linuxNS,
	}

	nseconnect.RegisterEndpointConnectionServer(grpcServer, nseConn)
	go func() {
		wg.Add(1)
		if err := grpcServer.Serve(connectionServer); err != nil {
			logrus.Fatalf("nse: failed to start grpc server on socket %s with error: %+v ", connectionServerSocket, err)
		}
	}()
	// Check if the socket of Endpoint Connection Server is operation
	testSocket, err := tools.SocketOperationCheck(connectionServerSocket)
	if err != nil {
		logrus.Fatalf("nse: failure to communicate with the socket %s with error: %+v", connectionServerSocket, err)
	}
	testSocket.Close()

	// NSE connection server is ready and now endpoints can be advertised to NSM
	advertiseSocket := path.Join(EndpointSocketBaseDir, EndpointSocket)

	if _, err := os.Stat(advertiseSocket); err != nil {
		logrus.Errorf("nse: failure to access nsm socket at %s with error: %+v, exiting...", advertiseSocket, err)
		os.Exit(1)
	}

	conn, err := tools.SocketOperationCheck(advertiseSocket)
	if err != nil {
		logrus.Fatalf("nse: failure to communicate with the socket %s with error: %+v", advertiseSocket, err)
	}
	defer conn.Close()
	logrus.Infof("nsm: connection to nsm server on socket: %s succeeded.", advertiseSocket)

	advertieConnection := nseconnect.NewEndpointOperationsClient(conn)

	endpoint := netmesh.NetworkServiceEndpoint{
		NseProviderName:    linuxNS,
		NetworkServiceName: networkServiceName,
		SocketLocation:     connectionServerSocket,
		LocalMechanisms: []*common.LocalMechanism{
			{
				Type: common.LocalMechanismType_KERNEL_INTERFACE,
			},
			{
				Type: common.LocalMechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					nsmutils.NSMPerPodDirectory: perPodDirectory,
				},
			},
		},
	}
	resp, err := advertieConnection.AdvertiseEndpoint(context.Background(), &nseconnect.EndpointAdvertiseRequest{
		RequestId:       linuxNS,
		NetworkEndpoint: &endpoint,
	})
	if err != nil {
		grpcServer.Stop()
		logrus.Fatalf("nse: failure to communicate with the socket %s with error: %+v", advertiseSocket, err)

	}
	if !resp.Accepted {
		grpcServer.Stop()
		logrus.Fatalf("nse: NSM response is inidcating failure of accepting endpoint Advertisiment.")
	}

	logrus.Infof("nse: channel has been successfully advertised, waiting for connection from NSM...")
	// Now block on WaitGroup
	wg.Wait()
}
