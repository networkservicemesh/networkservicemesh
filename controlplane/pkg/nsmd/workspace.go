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
	"github.com/networkservicemesh/networkservicemesh/security/manager"
	"context"
	"net"
	"os"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/local"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nseregistry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
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
	registryServer          NSERegistryServer
	networkServiceServer    networkservice.NetworkServiceServer
	monitorConnectionServer local.MonitorServer
	grpcServer              *grpc.Server
	sync.Mutex
	state            WorkspaceState
	locationProvider serviceregistry.WorkspaceLocationProvider
	localRegistry    *nseregistry.NSERegistry
}

func NewWorkSpace(nsm *nsmServer, name string, restore bool) (*Workspace, error) {
	logrus.Infof("Creating new workspace: %s", name)
	w := &Workspace{
		locationProvider: nsm.locationProvider,
		name:             name,
		state:            NEW,
		localRegistry:    nsm.localRegistry,
	}
	defer w.cleanup() // Cleans up if and only iff we are not in state RUNNING
	if !restore {
		if err := w.clearContents(); err != nil {
			return nil, err
		}
	}
	logrus.Infof("Creating new directory: %s", w.NsmDirectory())
	if err := os.MkdirAll(w.NsmDirectory(), os.ModePerm); err != nil {
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
	w.registryServer = NewRegistryServer(nsm, w)

	logrus.Infof("Creating new MonitorConnectionServer")
	w.monitorConnectionServer = local.NewMonitorServer()

	logrus.Infof("Creating new NetworkServiceServer")
	w.networkServiceServer = NewNetworkServiceServer(nsm.model, w, nsm.manager, nsm.serviceRegistry)

	logrus.Infof("Creating new GRPC MonitorServer")
	tracer := opentracing.GlobalTracer()
	cm := security.NewCertificateManager()
	cred, err := cm.ServerCredentials()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	w.grpcServer = grpc.NewServer(
		grpc.Creds(cred),
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	logrus.Infof("Registering NetworkServiceRegistryServer with registerServer")
	registry.RegisterNetworkServiceRegistryServer(w.grpcServer, w.registryServer)
	logrus.Infof("Registering NetworkServiceServer with registerServer")
	networkservice.RegisterNetworkServiceServer(w.grpcServer, w.networkServiceServer)
	logrus.Infof("Registering MonitorConnectionServer with registerServer")
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
	//conn, err := tools.SocketOperationCheck(tools.SocketPath(socket))
	//if err != nil {
	//	logrus.Errorf("failure to communicate with the socket %s with error: %+v", socket, err)
	//	return nil, err
	//}
	//conn.Close()
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

// MonitorConnectionServer returns workspace.monitorConnectionServer
func (w *Workspace) MonitorConnectionServer() local.MonitorServer {
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

func (w *Workspace) isConnectionAlive(timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	nseConn, err := tools.SocketOperationCheckContext(ctx, tools.SocketPath(w.NsmClientSocket()))
	if err != nil {
		return false
	}
	_ = nseConn.Close()

	return true
}

func (w *Workspace) cleanup() {
	if w.state != RUNNING {
		if w.NsmDirectory() != "" {
			w.clearContents()
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

func (w *Workspace) clearContents() error {
	if _, err := os.Stat(w.NsmDirectory()); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	logrus.Infof("Removing exist content im %s", w.NsmDirectory())
	err := os.RemoveAll(w.NsmDirectory())
	return err
}
