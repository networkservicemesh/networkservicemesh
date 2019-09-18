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
	"context"
	"fmt"
	"net"
	"path"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	dataplaneapi "github.com/networkservicemesh/networkservicemesh/dataplane/api/dataplane"
	dataplaneregistrarapi "github.com/networkservicemesh/networkservicemesh/dataplane/api/dataplaneregistrar"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
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
	model                        model.Model
	grpcServer                   *grpc.Server
	dataplaneRegistrarSocketPath string
	sock                         net.Listener
}

// dataplaneMonitor is per registered dataplane monitoring routine. It creates a grpc client
// for the socket advertsied by the dataplane and listens for a stream of operational Parameters/Constraints changes.
// All changes are reflected in the corresponding dataplane object in the object store.
// If it detects a failure of the connection, it will indicate that dataplane is no longer operational. In this case
// dataplaneMonitor will remove dataplane object from the object store and will terminate itself.
func dataplaneMonitor(model model.Model, dataplaneName string) {
	var err error
	dataplane := model.GetDataplane(dataplaneName)
	if dataplane == nil {
		logrus.Errorf("Dataplane object store does not have registered plugin %s", dataplaneName)
		return
	}
	conn, err := tools.DialUnix(dataplane.SocketLocation)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", dataplane.SocketLocation, err)
		model.DeleteDataplane(dataplaneName)
		return
	}
	defer conn.Close()
	dataplaneClient := dataplaneapi.NewMechanismsMonitorClient(conn)

	// Looping indefinitely or until grpc returns an error indicating the other end closed connection.
	stream, err := dataplaneClient.MonitorMechanisms(context.Background(), &empty.Empty{})
	if err != nil {
		logrus.Errorf("fail to create update grpc channel for Dataplane %s with error: %+v, removing dataplane from Objectstore.", dataplane.RegisteredName, err)
		model.DeleteDataplane(dataplaneName)
		return
	}
	for {
		updates, err := stream.Recv()
		if err != nil {
			logrus.Errorf("fail to receive on update grpc channel for Dataplane %s with error: %+v, removing dataplane from Objectstore.", dataplane.RegisteredName, err)
			model.DeleteDataplane(dataplaneName)
			return
		}
		logrus.Infof("Dataplane %s informed of its parameters changes, applying new parameters %+v", dataplaneName, updates.RemoteMechanisms)
		// TODO: this is not good -- direct model changes
		dataplane.SetRemoteMechanisms(updates.RemoteMechanisms)
		dataplane.SetLocalMechanisms(updates.LocalMechanisms)
		dataplane.MechanismsConfigured = true
		model.UpdateDataplane(dataplane)
	}
}

// RequestLiveness is a stream initiated by NSM to inform the dataplane that NSM is still alive and
// no re-registration is required. Detection a failure on this "channel" will mean
// that NSM is gone and the dataplane needs to start re-registration logic.
func (r *dataplaneRegistrarServer) RequestLiveness(liveness dataplaneregistrarapi.DataplaneRegistration_RequestLivenessServer) error {
	logrus.Infof("Liveness Request received")
	for {
		if err := liveness.SendMsg(&empty.Empty{}); err != nil {
			logrus.Errorf("deteced error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
			return err
		}
		time.Sleep(time.Second * livenessInterval)
	}
}

func (r *dataplaneRegistrarServer) RequestDataplaneRegistration(ctx context.Context, req *dataplaneregistrarapi.DataplaneRegistrationRequest) (*dataplaneregistrarapi.DataplaneRegistrationReply, error) {
	logrus.Infof("Received new dataplane registration requests from %s", req.DataplaneName)
	// Need to check if name of dataplane already exists in the object store
	if r.model.GetDataplane(req.DataplaneName) != nil {
		logrus.Errorf("dataplane with name %s already exist", req.DataplaneName)
		// TODO (sbezverk) Need to decide the right action, fail or not, failing for now
		return &dataplaneregistrarapi.DataplaneRegistrationReply{Registered: false}, fmt.Errorf("dataplane with name %s already registered", req.DataplaneName)
	}
	// Instantiating dataplane object with parameters from the request and creating a new object in the Object store
	dataplane := &model.Dataplane{
		RegisteredName: req.DataplaneName,
		SocketLocation: req.DataplaneSocket,
	}

	r.model.AddDataplane(dataplane)

	// Starting per dataplane go routine which will open grpc client connection on dataplane advertised socket
	// and will listen for operational parameters/constraints changes and reflecting these changes in the dataplane
	// object.
	go dataplaneMonitor(r.model, req.DataplaneName)

	return &dataplaneregistrarapi.DataplaneRegistrationReply{Registered: true}, nil
}

func (r *dataplaneRegistrarServer) RequestDataplaneUnRegistration(ctx context.Context, req *dataplaneregistrarapi.DataplaneUnRegistrationRequest) (*dataplaneregistrarapi.DataplaneUnRegistrationReply, error) {
	logrus.Infof("Received dataplane un-registration requests from %s", req.DataplaneName)

	// Removing dataplane from the store, if it does not exists, it does not matter as long as it is no longer there.
	r.model.DeleteDataplane(req.DataplaneName)

	return &dataplaneregistrarapi.DataplaneUnRegistrationReply{UnRegistered: true}, nil
}

// startDataplaneServer starts for a server listening for local NSEs advertise/remove
// dataplane registrar calls
func (dataplaneRegistrarServer *dataplaneRegistrarServer) startDataplaneRegistrarServer() error {
	dataplaneRegistrar := dataplaneRegistrarServer.dataplaneRegistrarSocketPath
	if err := tools.SocketCleanup(dataplaneRegistrar); err != nil {
		return err
	}

	unix.Umask(socketMask)
	var err error
	dataplaneRegistrarServer.sock, err = net.Listen("unix", dataplaneRegistrar)
	if err != nil {
		logrus.Errorf("failure to listen on socket %s with error: %+v", dataplaneRegistrar, err)
		return err
	}

	// Plugging dataplane registrar operations methods
	dataplaneregistrarapi.RegisterDataplaneRegistrationServer(dataplaneRegistrarServer.grpcServer, dataplaneRegistrarServer)
	// Plugging dataplane registrar operations methods
	dataplaneregistrarapi.RegisterDataplaneUnRegistrationServer(dataplaneRegistrarServer.grpcServer, dataplaneRegistrarServer)

	logrus.Infof("Starting Dataplane Registrar gRPC server listening on socket: %s", dataplaneRegistrar)
	go func() {
		if err := dataplaneRegistrarServer.grpcServer.Serve(dataplaneRegistrarServer.sock); err != nil {
			logrus.Fatalln("unable to start dataplane registrar grpc server: ", dataplaneRegistrar, err)
		}
	}()

	conn, err := tools.DialUnix(dataplaneRegistrar)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", dataplaneRegistrar, err)
		return err
	}
	_ = conn.Close()
	logrus.Infof("dataplane registrar Server socket: %s is operational", dataplaneRegistrar)

	return nil
}

func (dataplaneRegistrarServer *dataplaneRegistrarServer) Stop() {
	dataplaneRegistrarServer.grpcServer.GracefulStop()
	_ = dataplaneRegistrarServer.sock.Close()
}

// StartDataplaneRegistrarServer registers and starts gRPC server which is listening for
// Network Service Dataplane Registrar requests.
func StartDataplaneRegistrarServer(model model.Model) (*dataplaneRegistrarServer, error) {
	server := tools.NewServer()

	dataplaneRegistrarServer := &dataplaneRegistrarServer{
		grpcServer:                   server,
		dataplaneRegistrarSocketPath: path.Join(DataplaneRegistrarSocketBaseDir, DataplaneRegistrarSocket),
		model:                        model,
	}

	var err error
	// Starting dataplane registrar server, if it fails to start, inform Plugin by returning error
	err = dataplaneRegistrarServer.startDataplaneRegistrarServer()

	return dataplaneRegistrarServer, err
}
