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

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

type WorkspaceState int

const (
	NEW WorkspaceState = iota + 1
	RUNNING
	CLOSED
)

type tokenType string

const (
	workspaceTokenKey tokenType = "WorkspaceToken"
)

type Workspace struct {
	name       string
	listener   net.Listener
	grpcServer *grpc.Server
	sync.Mutex
	state            WorkspaceState
	locationProvider serviceregistry.WorkspaceLocationProvider
	ctx              context.Context
	token            string
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
		ctx:              span.Context(),
		token:            fmt.Sprintf("%s-%s", name, uuid.New().String()),
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

	if err := w.startServer(span, nsm); err != nil {
		return nil, err
	}

	span.Logger().Infof("Created new workspace: %+v", w)
	return w, nil
}

func (w *Workspace) startServer(span spanhelper.SpanHelper, nsm *nsmServer) error {
	socket := w.NsmServerSocket()
	span.Logger().Infof("Creating new listener on: %s", socket)
	listener, err := NewCustomListener(socket)
	if err != nil {
		span.LogError(err)
		return err
	}
	w.listener = listener

	span.Logger().Infof("Creating GRPC Server for the workspace")
	w.grpcServer = tools.NewServer(w.ctx,
		tools.WithChainedServerInterceptor(workspaceTokenExtractor()))
	w.registerServices(span, nsm)

	w.state = RUNNING
	go func() {
		defer w.Close()
		err = w.grpcServer.Serve(w.listener)
		if err != nil {
			span.Logger().Errorf("Serve() failed at workspace %+v: %v", w, err)
			return
		}
	}()

	// Check that the server we created responds
	conn, err := tools.DialUnix(socket)
	if err != nil {
		span.Logger().Errorf("failed to connect the socket at %s: %v", socket, err)
		return err
	}
	_ = conn.Close()

	span.Logger().Infof("gRPC server for workspace %s is operational", w.name)
	return nil
}

func (w *Workspace) registerServices(span spanhelper.SpanHelper, nsm *nsmServer) {
	span.Logger().Infof("Registering NetworkServiceRegistryServer with registerServer")
	registry.RegisterNetworkServiceRegistryServer(w.grpcServer, nsm.registryServer)
	span.Logger().Infof("Registering NetworkServiceDiscoveryServer with discoveryServer")
	registry.RegisterNetworkServiceDiscoveryServer(w.grpcServer, nsm.discoveryServer)
	span.Logger().Infof("Registering NetworkServiceServer with registerServer")
	unified.RegisterNetworkServiceServer(w.grpcServer, nsm.localServiceServer)
	span.Logger().Infof("Registering MonitorConnectionServer with registerServer")
	connection.RegisterMonitorConnectionServer(w.grpcServer, nsm.connectionMonitorServer)
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

// Token returns string token generated by NSM for the workspace
func (w *Workspace) Token() string {
	return w.token
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

// WithWorkspaceToken - creates a context with given workspace token
func WithWorkspaceToken(parent context.Context, token string) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, workspaceTokenKey, token)
}

// WorkspaceToken - extracts a workspace token form the context
func WorkspaceToken(ctx context.Context) string {
	value := ctx.Value(workspaceTokenKey)
	if value == nil {
		return ""
	}
	return value.(string)
}

// Extracts workspace token from incoming request headers and adds it to request's context
func workspaceTokenExtractor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if tokens := md.Get(common.WorkspaceTokenHeader); len(tokens) > 0 {
				ctx = WithWorkspaceToken(ctx, tokens[0])
			}
		}

		return handler(ctx, req)
	}
}
