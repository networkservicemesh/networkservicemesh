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

package nsmd

import (
	"fmt"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"net"
	"path"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
)

const (
	// EndpointSocketBaseDir defines the location of NSM Endpoints listen socket
	EndpointSocketBaseDir = "/var/lib/networkservicemesh"
	// EndpointSocket defines the name of NSM Endpoints operations socket
	EndpointSocket = "nsm.endpoint.io.sock"
	// EndpointServiceLabel is a label which is used to select all Endpoint object
	// for a specific Network Service
	EndpointServiceLabel = "networkservicemesh.io/network-service-name"
)

type nsmEndpointServer struct {
	model              model.Model
	grpcServer         *grpc.Server
	endPointSocketPath string
	stopChannel        chan bool
	nsmNamespace       string
	nsmPodIPAddress    string
}

func (es nsmEndpointServer) RegisterNSE(ctx context.Context,
	request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	logrus.Infof("Received RegisterNSE request: %v", request)

	// Check if there is already Network Service Endpoint object with the same name, if there is
	// success will be returned to NSE, since it is a case of NSE pod coming back up.
	ep := es.model.GetEndpoint(request.EndpointName)
	if ep != nil {
		return nil, fmt.Errorf("Network Service Endpoint object %s already exists", request.EndpointName)
	}

	es.model.AddEndpoint(request)
	return request, nil
}

func (es nsmEndpointServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*common.Empty, error) {
	logrus.Infof("Received Endpoint Remove request: %+v", request)
	if err := es.model.DeleteEndpoint(request.EndpointName); err != nil {
		return &common.Empty{}, err
	}
	return &common.Empty{}, nil
}

// startEndpointServer starts for a server listening for local NSEs advertise/remove
// endpoint calls
func startEndpointServer(endpointServer *nsmEndpointServer) error {
	listenEndpoint := endpointServer.endPointSocketPath
	if err := tools.SocketCleanup(listenEndpoint); err != nil {
		return err
	}

	unix.Umask(socketMask)
	sock, err := net.Listen("unix", listenEndpoint)
	if err != nil {
		logrus.Errorf("failure to listen on socket %s with error: %+v", listenEndpoint, err)
		return err
	}

	// Plugging Endpoint operations methods
	registry.RegisterNetworkServiceRegistryServer(endpointServer.grpcServer, endpointServer)
	logrus.Infof("Starting Endpoint gRPC server listening on socket: %s", listenEndpoint)
	go func() {
		if err := endpointServer.grpcServer.Serve(sock); err != nil {
			logrus.Fatalln("unable to start endpoint grpc server: ", listenEndpoint, err)
		}
	}()

	conn, err := tools.SocketOperationCheck(listenEndpoint)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", listenEndpoint, err)
		return err
	}
	conn.Close()
	logrus.Infof("Endpoint Server socket: %s is operational", listenEndpoint)

	// Wait for shutdown
	select {
	case <-endpointServer.stopChannel:
		logrus.Infof("Server for socket %s received shutdown request", listenEndpoint)
	}
	endpointServer.stopChannel <- true

	return nil
}

// StartEndpointServer registers and starts gRPC server which is listening for
// Network Service Endpoint advertise/remove calls and act accordingly
func StartEndpointServer(model model.Model) error {
	endpointServer := &nsmEndpointServer{
		model:              model,
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
