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

package nsmserver

import (
	"net"
	"path"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nseconnect"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/networkservicemesh/plugins/k8sclient"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
)

const (
	// EndpointSocketBaseDir defines the location of NSM Endpoints listen socket
	EndpointSocketBaseDir = "/var/lib/networkservicemesh"
	// EndpointSocket defines the name of NSM Endpoints operations socket
	EndpointSocket = "nsm.endpoint.io.sock"
)

type nsmEndpointServer struct {
	logger             logger.FieldLoggerPlugin
	objectStore        objectstore.Interface
	client             k8sclient.PluginAPI
	grpcServer         *grpc.Server
	endPointSocketPath string
	stopChannel        chan bool
}

func (e nsmEndpointServer) AdvertiseEndpoint(ctx context.Context,
	ar *nseconnect.EndpointAdvertiseRequest) (*nseconnect.EndpointAdvertiseReply, error) {
	e.logger.Infof("Received Endpoint Advertise request: %+v", ar)
	return &nseconnect.EndpointAdvertiseReply{
		RequestId: ar.RequestId,
		Accepted:  true,
	}, nil
}

func (e nsmEndpointServer) RemoveEndpoint(ctx context.Context,
	rr *nseconnect.EndpointRemoveRequest) (*nseconnect.EndpointRemoveReply, error) {
	e.logger.Infof("Received Endpoint Remove request: %+v", rr)
	return &nseconnect.EndpointRemoveReply{
		RequestId: rr.RequestId,
		Accepted:  true,
	}, nil
}

// startEndpointServer starts for a server listening for local NSEs advertise/remove
// endpoint calls
func startEndpointServer(endpointServer *nsmEndpointServer) error {
	listenEndpoint := endpointServer.endPointSocketPath
	logger := endpointServer.logger
	if err := tools.SocketCleanup(listenEndpoint); err != nil {
		return err
	}

	unix.Umask(socketMask)
	sock, err := net.Listen("unix", listenEndpoint)
	if err != nil {
		logger.Errorf("failure to listen on socket %s with error: %+v", listenEndpoint, err)
		return err
	}

	// Plugging Endpoint operations methods
	nseconnect.RegisterEndpointOperationsServer(endpointServer.grpcServer, endpointServer)
	logger.Infof("Starting Endpoint gRPC server listening on socket: %s", listenEndpoint)
	go func() {
		if err := endpointServer.grpcServer.Serve(sock); err != nil {
			logger.Fatalln("unable to start endpoint grpc server: ", listenEndpoint, err)
		}
	}()

	conn, err := tools.SocketOperationCheck(listenEndpoint)
	if err != nil {
		logger.Errorf("failure to communicate with the socket %s with error: %+v", listenEndpoint, err)
		return err
	}
	conn.Close()
	logger.Infof("Endpoint Server socket: %s is operational", listenEndpoint)

	// Wait for shutdown
	select {
	case <-endpointServer.stopChannel:
		logger.Infof("Server for socket %s received shutdown request", listenEndpoint)
	}
	endpointServer.stopChannel <- true

	return nil
}

// NewNSMEndpointServer registers and starts gRPC server which is listening for
// Network Service Endpoint advertise/remove calls and act accordingly
func NewNSMEndpointServer(p *Plugin) error {
	endpointServer := &nsmEndpointServer{
		logger:             p.Deps.Log,
		objectStore:        p.Deps.ObjectStore,
		client:             p.Deps.Client,
		grpcServer:         grpc.NewServer(),
		endPointSocketPath: path.Join(EndpointSocketBaseDir, EndpointSocket),
		stopChannel:        make(chan bool),
	}

	var err error
	// Starting endpoint server, if it fails to start, inform Plugin by returning error
	go func() {
		err = startEndpointServer(endpointServer)
	}()

	return err
}
