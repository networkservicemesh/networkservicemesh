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

package dataplaneregistrar

import (
	"context"
	"net"
	"path"

	dataplaneregistrarapi "github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplaneregistrar"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
)

const (
	// DataplaneRegistrarSocketBaseDir defines the location of NSM dataplane registrar listen socket
	DataplaneRegistrarSocketBaseDir = "/var/lib/networkservicemesh"
	// DataplaneRegistrarSocket defines the name of NSM dataplane registrar socket
	DataplaneRegistrarSocket = "nsm.dataplane-registrar.io.sock"
	socketMask               = 0077
)

type dataplaneRegistrarServer struct {
	logger                       logger.FieldLoggerPlugin
	objectStore                  objectstore.Interface
	grpcServer                   *grpc.Server
	dataplaneRegistrarSocketPath string
	stopChannel                  chan bool
}

func (r *dataplaneRegistrarServer) RequestDataplaneRegistration(ctx context.Context, req *dataplaneregistrarapi.DataplaneRegistrationRequest) (*dataplaneregistrarapi.DataplaneRegistrationReply, error) {

	return &dataplaneregistrarapi.DataplaneRegistrationReply{Registred: true}, nil
}

// startDataplaneServer starts for a server listening for local NSEs advertise/remove
// dataplane registrar calls
func startDataplaneRegistrarServer(dataplaneRegistrarServer *dataplaneRegistrarServer) error {
	dataplaneRegistrar := dataplaneRegistrarServer.dataplaneRegistrarSocketPath
	logger := dataplaneRegistrarServer.logger
	if err := tools.SocketCleanup(dataplaneRegistrar); err != nil {
		return err
	}

	unix.Umask(socketMask)
	sock, err := net.Listen("unix", dataplaneRegistrar)
	if err != nil {
		logger.Errorf("failure to listen on socket %s with error: %+v", dataplaneRegistrar, err)
		return err
	}

	// Plugging dataplane registrar operations methods
	dataplaneregistrarapi.RegisterDataplaneRegistrationServer(dataplaneRegistrarServer.grpcServer, dataplaneRegistrarServer)

	logger.Infof("Starting Dataplane Registrar gRPC server listening on socket: %s", dataplaneRegistrar)
	go func() {
		if err := dataplaneRegistrarServer.grpcServer.Serve(sock); err != nil {
			logger.Fatalln("unable to start dataplane registrar grpc server: ", dataplaneRegistrar, err)
		}
	}()

	conn, err := tools.SocketOperationCheck(dataplaneRegistrar)
	if err != nil {
		logger.Errorf("failure to communicate with the socket %s with error: %+v", dataplaneRegistrar, err)
		return err
	}
	conn.Close()
	logger.Infof("dataplane registrar Server socket: %s is operational", dataplaneRegistrar)

	// Wait for shutdown
	select {
	case <-dataplaneRegistrarServer.stopChannel:
		logger.Infof("Server for socket %s received shutdown request", dataplaneRegistrar)
	}
	dataplaneRegistrarServer.stopChannel <- true

	return nil
}

// NewDataplaneRegistrarServer registers and starts gRPC server which is listening for
// Network Service Dataplane Registrar requests.
func NewDataplaneRegistrarServer(p *Plugin) error {
	dataplaneRegistrarServer := &dataplaneRegistrarServer{
		logger:                       p.Deps.Log,
		objectStore:                  p.Deps.ObjectStore,
		grpcServer:                   grpc.NewServer(),
		dataplaneRegistrarSocketPath: path.Join(DataplaneRegistrarSocketBaseDir, DataplaneRegistrarSocket),
		stopChannel:                  make(chan bool),
	}

	var err error
	// Starting dataplane registrar server, if it fails to start, inform Plugin by returning error
	go func() {
		err = startDataplaneRegistrarServer(dataplaneRegistrarServer)
	}()

	return err
}
