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
	"os"
	"sync"
	"time"

	connectionMonitor "github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nseregistry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
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
	monitorConnectionServer connectionMonitor.MonitorServer
	grpcServer              *grpc.Server
	sync.Mutex
	state            WorkspaceState
	locationProvider serviceregistry.WorkspaceLocationProvider
	localRegistry    *nseregistry.NSERegistry
	ctx              context.Context
	discoveryServer  registry.NetworkServiceDiscoveryServer
}

// NewWorkSpace - constructs a new workspace.
func NewWorkSpace(ctx context.Context, nsm *nsmServer, name string, restore bool) (*Workspace, error) {
	span := spanhelper.FromContext(ctx, fmt.Sprintf("Workspace:%v", name))
	defer span.Finish()
	span.Logger().Infof("Creating new workspace: %s", name)
	w := &Workspace{
		locationProvider: nsm.locationProvider,
		name:             name,
		state:            NEW,
		localRegistry:    nsm.localRegistry,
		ctx:              span.Context(),
	}
	defer w.cleanup() // Cleans up if and only iff we are not in state RUNNING
	span.LogValue("restore", restore)
	if !restore {
		if err := w.clearContents(span.Context()); err != nil {
			return nil, err
		}
	}
	span.Logger().Infof("Creating new directory: %s", w.NsmDirectory())
	if err := os.MkdirAll(w.NsmDirectory(), os.ModePerm); err != nil {
		span.Logger().Errorf("can't create folder: %s, error: %v", w.NsmDirectory(), err)
		return nil, err
	}
	socket := w.NsmServerSocket()
	span.Logger().Infof("Creating new listener on: %s", socket)
	listener, err := NewCustomListener(socket)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	w.listener = listener

	registerWorkspaceServices(span, w, nsm)
	w.state = RUNNING
	go func() {
		defer w.Close()
		err = w.grpcServer.Serve(w.listener)
		if err != nil {
			span.Logger().Errorf("Failed to server workspace %+v: %s", w, err)
			return
		}
	}()

	conn, err := tools.DialUnix(socket)
	if err != nil {
		span.Logger().Errorf("failure to communicate with the socket %s with error: %+v", socket, err)
		return nil, err
	}
	_ = conn.Close()
	span.Logger().Infof("grpcserver for workspace %+v is operational", w)
	span.Logger().Infof("Created new workspace: %+v", w)
	return w, nil
}

func registerWorkspaceServices(span spanhelper.SpanHelper, w *Workspace, nsm *nsmServer) {
	span.Logger().Infof("Creating new NetworkServiceRegistryServer")
	w.registryServer = NewRegistryServer(nsm, w)
	w.discoveryServer = NewNetworkServiceDiscoveryServer(nsm.serviceRegistry)

	span.Logger().Infof("Creating new MonitorConnectionServer")
	w.monitorConnectionServer = connectionMonitor.NewMonitorServer("LocalConnection")

	span.Logger().Infof("Creating new NetworkServiceServer")
	w.networkServiceServer = NewNetworkServiceServer(nsm.model, w, nsm.manager)

	span.Logger().Infof("Creating new GRPC MonitorServer")
	w.grpcServer = tools.NewServer(span.Context())

	span.Logger().Infof("Registering NetworkServiceRegistryServer with registerServer")
	registry.RegisterNetworkServiceRegistryServer(w.grpcServer, w.registryServer)
	span.Logger().Infof("Registering NetworkServiceDiscoveryServer with discoveryServer")
	registry.RegisterNetworkServiceDiscoveryServer(w.grpcServer, w.discoveryServer)
	span.Logger().Infof("Registering NetworkServiceServer with registerServer")
	unified.RegisterNetworkServiceServer(w.grpcServer, w.networkServiceServer)
	span.Logger().Infof("Registering MonitorConnectionServer with registerServer")
	connection.RegisterMonitorConnectionServer(w.grpcServer, w.monitorConnectionServer)
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
func (w *Workspace) MonitorConnectionServer() connectionMonitor.MonitorServer {
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

func (w *Workspace) isConnectionAlive(ctx context.Context, timeout time.Duration) bool {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	span := spanhelper.CopySpan(timeoutCtx, spanhelper.GetSpanHelper(ctx), "check-nse-alive")
	defer span.Finish()

	nseConn, err := tools.DialContextUnix(timeoutCtx, w.NsmClientSocket())
	if err != nil {
		span.LogObject("alive", false)
		return false
	}
	_ = nseConn.Close()
	span.LogObject("alive", true)
	return true
}

func (w *Workspace) cleanup() {
	span := spanhelper.FromContext(w.ctx, "cleanup")
	defer span.Finish()
	if w.state != RUNNING {
		if w.NsmDirectory() != "" {
			err := w.clearContents(w.ctx)
			span.LogError(err)
		}
		if w.grpcServer != nil {
			// TODO switch to Graceful stop once we think through possible long running connections
			w.grpcServer.Stop()
		}
		if w.listener != nil {
			err := w.listener.Close()
			span.LogError(err)
		}
	}
}

func (w *Workspace) clearContents(ctx context.Context) error {
	span := spanhelper.FromContext(ctx, "clearContents")
	defer span.Finish()
	if _, err := os.Stat(w.NsmDirectory()); err != nil {
		if os.IsNotExist(err) {
			span.Logger().Infof("No exist folder %s", w.NsmDirectory())
			return nil
		}
		span.LogError(err)
		return err
	}
	span.Logger().Infof("Removing exist content im %s", w.NsmDirectory())
	err := os.RemoveAll(w.NsmDirectory())
	return err
}
