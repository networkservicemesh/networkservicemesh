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
	"fmt"
	"net"
	"path"
	"time"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplaneinterface"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	dataplaneregistrarapi "github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplaneregistrar"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	// DataplaneRegistrarSocketBaseDir defines the location of NSM dataplane registrar listen socket
	DataplaneRegistrarSocketBaseDir = "/var/lib/networkservicemesh"
	// DataplaneRegistrarSocket defines the name of NSM dataplane registrar socket
	DataplaneRegistrarSocket = "nsm.dataplane-registrar.io.sock"
	socketMask               = 0077
	livenessInterval         = 5
)

type dataplaneRegistrarServer struct {
	logger                       logger.FieldLoggerPlugin
	objectStore                  objectstore.Interface
	grpcServer                   *grpc.Server
	dataplaneRegistrarSocketPath string
	stopChannel                  chan bool
}

// dataplaneMonitor is per registered dataplane monitoring routine. It creates a grpc client
// for the socket advertsied by the dataplane and listens for a stream of operational Parameters/Constraints changes.
// All changes are reflected in the corresponding dataplane object in the object store.
// If it detects a failure of the connection, it will indicate that dataplane is no longer operational. In this case
// dataplaneMonitor will remove dataplane object from the object store and will terminate itself.
func dataplaneMonitor(objStore objectstore.Interface, dataplaneName string, logger logger.FieldLoggerPlugin) {
	var err error
	dataplane := objStore.GetDataplane(dataplaneName)
	if dataplane == nil {
		logger.Errorf("Dataplane object store does not have registered plugin %s", dataplaneName)
		return
	}
	dataplane.Conn, err = tools.SocketOperationCheck(dataplane.SocketLocation)
	if err != nil {
		logger.Errorf("failure to communicate with the socket %s with error: %+v", dataplane.SocketLocation, err)
		objStore.ObjectDeleted(&dataplaneName)
		return
	}
	defer dataplane.Conn.Close()
	dataplane.DataplaneClient = dataplaneinterface.NewDataplaneOperationsClient(dataplane.Conn)

	// Looping indefinetly or until grpc returns an error indicating the other end closed connection.
	stream, err := dataplane.DataplaneClient.UpdateDataplane(context.Background(), &common.Empty{})
	if err != nil {
		logger.Errorf("fail to create update grpc channel for Dataplane %s with error: %+v, removing dataplane from Objectstore.", dataplane.RegisteredName, err)
		objStore.ObjectDeleted(&dataplaneName)
		return
	}
	for {
		updates, err := stream.Recv()
		if err != nil {
			logger.Errorf("fail to receive on update grpc channel for Dataplane %s with error: %+v, removing dataplane from Objectstore.", dataplane.RegisteredName, err)
			objStore.ObjectDeleted(dataplane)
			return
		}
		logger.Infof("Dataplane %s informed of its parameters changes, applying new parameters %+v", updates.RemoteMechanism)
		// TODO (sbezverk) Apply changes received from dataplane onto the corresponding dataplane object in the Object store
	}
}

// RequestLiveness is a stream initiated by NSM to inform the dataplane that NSM is still alive and
// no re-registration is required. Detection a failure on this "channel" will mean
// that NSM is gone and the dataplane needs to start re-registration logic.
func (r *dataplaneRegistrarServer) RequestLiveness(liveness dataplaneregistrarapi.DataplaneRegistration_RequestLivenessServer) error {
	r.logger.Infof("Liveness Request received")
	for {
		if err := liveness.SendMsg(&common.Empty{}); err != nil {
			r.logger.Errorf("deteced error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
			return err
		}
		time.Sleep(time.Second * livenessInterval)
	}
}

func (r *dataplaneRegistrarServer) RequestDataplaneRegistration(ctx context.Context, req *dataplaneregistrarapi.DataplaneRegistrationRequest) (*dataplaneregistrarapi.DataplaneRegistrationReply, error) {
	r.logger.Infof("Received new dataplane registration requests from %s", req.DataplaneName)
	// Need to check if name of dataplane already exists in the object store
	if r.objectStore.GetDataplane(req.DataplaneName) != nil {
		r.logger.Errorf("dataplane with name %s already exist", req.DataplaneName)
		// TODO (sbezverk) Need to decide the right action, fail or not, failing for now
		return &dataplaneregistrarapi.DataplaneRegistrationReply{Registered: false}, fmt.Errorf("dataplane with name %s already registered", req.DataplaneName)
	}
	// Instantiating dataplane object with parameters from the request and creating a new object in the Object store
	dataplane := &objectstore.Dataplane{
		RegisteredName:     req.DataplaneName,
		SocketLocation:     req.DataplaneSocket,
		RemoteMechanism:    req.RemoteMechanism,
		SupportedInterface: req.SupportedInterface,
	}
	r.objectStore.ObjectCreated(dataplane)
	// Starting per dataplane go routine which will open grpc client connection on dataplane advertised socket
	// and will listen for operational parameters/constraints changes and reflecting these changes in the dataplane
	// object.
	go dataplaneMonitor(r.objectStore, req.DataplaneName, r.logger)

	return &dataplaneregistrarapi.DataplaneRegistrationReply{Registered: true}, nil
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
