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
	"net"
	"os"
	"strings"
	"sync"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/ligato/networkservicemesh/pkg/tools"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type WorkspaceState int

const (
	NEW WorkspaceState = iota + 1
	RUNNING
	CLOSED
)

type Workspace struct {
	name                    string
	listener                net.Listener
	registryServer          registry.NetworkServiceRegistryServer
	networkServiceServer    networkservice.NetworkServiceServer
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
	grpcServer              *grpc.Server
	sync.Mutex
	state            WorkspaceState
	locationProvider serviceregistry.WorkspaceLocationProvider
}

func NewWorkSpace(model model.Model, serviceRegistry serviceregistry.ServiceRegistry, name string) (*Workspace, error) {
	logrus.Infof("Creating new workspace: %s", name)
	w := &Workspace{
		locationProvider: serviceRegistry.NewWorkspaceProvider(),
	}

	defer w.cleanup() // Cleans up if and only iff we are not in state RUNNING
	w.state = NEW
	w.name = name
	logrus.Infof("Creating new directory: %s", w.NsmDirectory())
	if err := os.MkdirAll(w.NsmDirectory(), folderMask); err != nil {
		logrus.Errorf("can't create folder: %s, error: %v", w.NsmDirectory(), err)
		return nil, err
	}
	socket := w.NsmServerSocket()
	logrus.Infof("Creating new listener on: %s", socket)
	listener, err := NewCustomListener(socket)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	w.listener = listener
	logrus.Infof("Creating new NetworkServiceRegistryServer")
	w.registryServer = NewRegistryServer(model, w, serviceRegistry)

	logrus.Infof("Creating new MonitorConnectionServer")
	w.monitorConnectionServer = monitor_connection_server.NewMonitorConnectionServer()

	logrus.Infof("Creating new NetworkServiceServer")
	w.networkServiceServer = NewNetworkServiceServer(model, w, serviceRegistry, getExcludedPrefixes())

	logrus.Infof("Creating new GRPC Server")
	tracer := opentracing.GlobalTracer()
	w.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	logrus.Infof("Registering NetworkServiceRegistryServer with grpcServer")
	registry.RegisterNetworkServiceRegistryServer(w.grpcServer, w.registryServer)
	logrus.Infof("Registering NetworkServiceServer with grpcServer")
	networkservice.RegisterNetworkServiceServer(w.grpcServer, w.networkServiceServer)
	logrus.Infof("Registering MonitorConnectionServer with grpcServer")
	connection.RegisterMonitorConnectionServer(w.grpcServer, w.monitorConnectionServer)
	w.state = RUNNING
	go func() {
		defer w.Close()
		err = w.grpcServer.Serve(w.listener)
		if err != nil {
			logrus.Errorf("Failed to server workspace %+v: %s", w, err)
			return
		}
	}()
	conn, err := tools.SocketOperationCheck(socket)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", socket, err)
		return nil, err
	}
	conn.Close()
	logrus.Infof("grpcserver for workspace %+v is operational", w)
	logrus.Infof("Created new workspace: %+v", w)
	return w, nil
}

func (w *Workspace) Name() string {
	return w.name
}

func (w *Workspace) NsmDirectory() string {
	return w.locationProvider.NsmBaseDir() + w.name
}

func (w *Workspace) HostDirectory() string {
	return w.locationProvider.NsmBaseDir() + w.name
}

func (w *Workspace) ClientDirectory() string {
	return w.locationProvider.ClientBaseDir()
}

func (w *Workspace) NsmServerSocket() string {
	return w.NsmDirectory() + "/" + w.locationProvider.NsmServerSocket()
}

func (w *Workspace) NsmClientSocket() string {
	return w.NsmDirectory() + "/" + w.locationProvider.NsmClientSocket()
}

func (w *Workspace) MonitorConnectionServer() monitor_connection_server.MonitorConnectionServer {
	if w == nil {
		return nil
	}
	return w.monitorConnectionServer
}

func (w *Workspace) Close() {
	// TODO handle cleanup here on failure in NewWorkspace creation
	w.Lock()
	defer w.Unlock()
	w.state = CLOSED
	w.cleanup()
}

func (w *Workspace) cleanup() {
	if w.state != RUNNING {
		if w.NsmDirectory() != "" {
			os.RemoveAll(w.NsmDirectory())
		}
		if w.grpcServer != nil {
			// TODO switch to Graceful stop once we think through possible long running connections
			w.grpcServer.Stop()
		}
		if w.listener != nil {
			w.listener.Close()
		}
	}
}

func getExcludedPrefixes() []string {
	//TODO: Add a better way to pass this value to NSMD
	excluded_prefixes := []string{}
	exclude_prefixes_env, ok := os.LookupEnv(ExcludedPrefixesEnv)
	if ok {
		for _, s := range strings.Split(exclude_prefixes_env, ",") {
			excluded_prefixes = append(excluded_prefixes, strings.TrimSpace(s))
		}
	}
	return excluded_prefixes
}
