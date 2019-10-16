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
	"net"
	"path"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	forwarderapi "github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	forwarderregistrarapi "github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarderregistrar"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	// DataplaneRegistrarSocketBaseDir defines the location of NSM forwarder registrar listen socket
	DataplaneRegistrarSocketBaseDir = "/var/lib/networkservicemesh"
	// DataplaneRegistrarSocket defines the name of NSM forwarder registrar socket
	DataplaneRegistrarSocket = "nsm.forwarder-registrar.io.sock"
	socketMask               = 0077
	livenessInterval         = 5
)

// DataplaneRegistrarServer - NSMgr registration service
type DataplaneRegistrarServer struct {
	model                        model.Model
	grpcServer                   *grpc.Server
	forwarderRegistrarSocketPath string
	sock                         net.Listener
}

// forwarderMonitor is per registered forwarder monitoring routine. It creates a grpc client
// for the socket advertsied by the forwarder and listens for a stream of operational Parameters/Constraints changes.
// All changes are reflected in the corresponding forwarder object in the object store.
// If it detects a failure of the connection, it will indicate that forwarder is no longer operational. In this case
// forwarderMonitor will remove forwarder object from the object store and will terminate itself.
func forwarderMonitor(model model.Model, forwarderName string) {
	var err error
	forwarder := model.GetDataplane(forwarderName)
	if forwarder == nil {
		logrus.Errorf("Dataplane object store does not have registered plugin %s", forwarderName)
		return
	}
	conn, err := tools.DialUnix(forwarder.SocketLocation)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", forwarder.SocketLocation, err)
		model.DeleteDataplane(context.Background(), forwarderName)
		return
	}
	defer conn.Close()
	forwarderClient := forwarderapi.NewMechanismsMonitorClient(conn)

	// Looping indefinitely or until grpc returns an error indicating the other end closed connection.
	stream, err := forwarderClient.MonitorMechanisms(context.Background(), &empty.Empty{})
	if err != nil {
		logrus.Errorf("fail to create update grpc channel for Dataplane %s with error: %+v, removing forwarder from Objectstore.", forwarder.RegisteredName, err)
		model.DeleteDataplane(context.Background(), forwarderName)
		return
	}
	for {
		updates, err := stream.Recv()
		if err != nil {
			logrus.Errorf("fail to receive on update grpc channel for Dataplane %s with error: %+v, removing forwarder from Objectstore.", forwarder.RegisteredName, err)
			model.DeleteDataplane(context.Background(), forwarderName)
			return
		}
		logrus.Infof("Dataplane %s informed of its parameters changes, applying new parameters %+v", forwarderName, updates.RemoteMechanisms)
		// TODO: this is not good -- direct model changes
		forwarder.SetRemoteMechanisms(updates.RemoteMechanisms)
		forwarder.SetLocalMechanisms(updates.LocalMechanisms)
		forwarder.MechanismsConfigured = true
		model.UpdateDataplane(context.Background(), forwarder)
	}
}

// RequestLiveness is a stream initiated by NSM to inform the forwarder that NSM is still alive and
// no re-registration is required. Detection a failure on this "channel" will mean
// that NSM is gone and the forwarder needs to start re-registration logic.
func (r *DataplaneRegistrarServer) RequestLiveness(liveness forwarderregistrarapi.DataplaneRegistration_RequestLivenessServer) error {
	logrus.Infof("Liveness Request received")
	for {
		if err := liveness.SendMsg(&empty.Empty{}); err != nil {
			logrus.Errorf("deteced error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
			return err
		}
		time.Sleep(time.Second * livenessInterval)
	}
}

// RequestDataplaneRegistration - request forwarder to be registered.
func (r *DataplaneRegistrarServer) RequestDataplaneRegistration(ctx context.Context, req *forwarderregistrarapi.DataplaneRegistrationRequest) (*forwarderregistrarapi.DataplaneRegistrationReply, error) {
	logrus.Infof("Received new forwarder registration requests from %s", req.DataplaneName)
	// Need to check if name of forwarder already exists in the object store
	if r.model.GetDataplane(req.DataplaneName) != nil {
		logrus.Errorf("forwarder with name %s already exist", req.DataplaneName)
		// TODO (sbezverk) Need to decide the right action, fail or not, failing for now
		return &forwarderregistrarapi.DataplaneRegistrationReply{Registered: false}, errors.Errorf("forwarder with name %s already registered", req.DataplaneName)
	}
	// Instantiating forwarder object with parameters from the request and creating a new object in the Object store
	forwarder := &model.Dataplane{
		RegisteredName: req.DataplaneName,
		SocketLocation: req.DataplaneSocket,
	}

	r.model.AddDataplane(ctx, forwarder)

	// Starting per forwarder go routine which will open grpc client connection on forwarder advertised socket
	// and will listen for operational parameters/constraints changes and reflecting these changes in the forwarder
	// object.
	go forwarderMonitor(r.model, req.DataplaneName)

	return &forwarderregistrarapi.DataplaneRegistrationReply{Registered: true}, nil
}

// RequestDataplaneUnRegistration - request forwarder to be unregistered
func (r *DataplaneRegistrarServer) RequestDataplaneUnRegistration(ctx context.Context, req *forwarderregistrarapi.DataplaneUnRegistrationRequest) (*forwarderregistrarapi.DataplaneUnRegistrationReply, error) {
	logrus.Infof("Received forwarder un-registration requests from %s", req.DataplaneName)

	// Removing forwarder from the store, if it does not exists, it does not matter as long as it is no longer there.
	r.model.DeleteDataplane(ctx, req.DataplaneName)

	return &forwarderregistrarapi.DataplaneUnRegistrationReply{UnRegistered: true}, nil
}

// startDataplaneServer starts for a server listening for local NSEs advertise/remove
// forwarder registrar calls
func (r *DataplaneRegistrarServer) startDataplaneRegistrarServer(ctx context.Context) error {
	span := spanhelper.FromContext(ctx, "StartDataplaneRegistrarServer")
	defer span.Finish()

	forwarderRegistrar := r.forwarderRegistrarSocketPath
	span.LogValue("path", forwarderRegistrar)
	if err := tools.SocketCleanup(forwarderRegistrar); err != nil {
		return err
	}

	unix.Umask(socketMask)
	var err error
	r.sock, err = net.Listen("unix", forwarderRegistrar)
	if err != nil {
		span.LogError(errors.WithMessagef(err, "failure to listen on socket %s", forwarderRegistrar))
		return err
	}

	// Plugging forwarder registrar operations methods
	forwarderregistrarapi.RegisterDataplaneRegistrationServer(r.grpcServer, r)
	// Plugging forwarder registrar operations methods
	forwarderregistrarapi.RegisterDataplaneUnRegistrationServer(r.grpcServer, r)

	span.Logger().Infof("Starting Dataplane Registrar gRPC server listening on socket: %s", forwarderRegistrar)
	go func() {
		if serverErr := r.grpcServer.Serve(r.sock); serverErr != nil {
			serverErr = errors.Errorf("unable to start forwarder registrar grpc server: %v %v", forwarderRegistrar, serverErr)
			span.LogError(serverErr)
			span.Logger().Fatalln(serverErr)
		}
	}()

	conn, dialErr := tools.DialContextUnix(span.Context(), forwarderRegistrar)
	if dialErr != nil {
		span.LogError(errors.Errorf("failure to communicate with the socket %s with error: %+v", forwarderRegistrar, dialErr))
		return err
	}
	_ = conn.Close()
	span.Logger().Infof("forwarder registrar Server socket: %s is operational", forwarderRegistrar)

	return nil
}

// Stop - stop forwarder registration socket.
func (r *DataplaneRegistrarServer) Stop() {
	r.grpcServer.Stop() // We do not need to do it gracefully, to speedup forwarder termination.
	_ = r.sock.Close()
}

// StartDataplaneRegistrarServer -  registers and starts gRPC server which is listening for
// Network Service Dataplane Registrar requests.
func StartDataplaneRegistrarServer(ctx context.Context, model model.Model) (*DataplaneRegistrarServer, error) {
	span := spanhelper.FromContext(ctx, "DataplaneRegistrarServer")
	defer span.Finish()
	server := tools.NewServer(span.Context())

	forwarderRegistrarServer := &DataplaneRegistrarServer{
		grpcServer:                   server,
		forwarderRegistrarSocketPath: path.Join(DataplaneRegistrarSocketBaseDir, DataplaneRegistrarSocket),
		model:                        model,
	}

	var err error
	// Starting forwarder registrar server, if it fails to start, inform Plugin by returning error
	err = forwarderRegistrarServer.startDataplaneRegistrarServer(span.Context())
	span.LogError(err)
	return forwarderRegistrarServer, err
}
