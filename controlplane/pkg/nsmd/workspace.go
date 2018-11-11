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
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type WorkspaceState int

const (
	NEW WorkspaceState = iota + 1
	RUNNING
	CLOSED
)

const (
	rootDir               = "/var/lib/networkservicemesh/"
	WorkspaceClientSocket = "nsm.client.io.sock"
	dirMask               = 0777
)

type Workspace struct {
	name                 string
	directory            string
	listener             net.Listener
	registryServer       registry.NetworkServiceRegistryServer
	networkServiceServer networkservice.NetworkServiceServer
	grpcServer           *grpc.Server
	sync.Mutex
	state WorkspaceState
}

func NewWorkSpace(model model.Model, name string) (*Workspace, error) {
	logrus.Infof("Creating new workspace: %s", name)
	w := &Workspace{}
	defer w.cleanup() // Cleans up if and only iff we are not in state RUNNING
	w.state = NEW
	w.name = name
	w.directory = rootDir + w.name
	logrus.Infof("Creating new directory: %s", w.directory)
	if err := os.MkdirAll(w.directory, folderMask); err != nil {
		logrus.Errorf("can't create folder: %s, error: %v", w.directory, err)
		return nil, err
	}
	socket := w.directory + "/" + WorkspaceClientSocket
	logrus.Infof("Creating new listener on: %s", socket)
	listener, err := NewCustomListener(socket)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	w.listener = listener
	logrus.Infof("Creating new NetworkServiceRegistryServer")
	w.registryServer = NewRegistryServer(model)
	logrus.Infof("Creating new NetworkServiceServer")
	w.networkServiceServer = NewNetworkServiceServer(model)

	logrus.Infof("Creating new GRPC Server")
	w.grpcServer = grpc.NewServer()
	logrus.Infof("Registering NetworkServiceRegistryServer with grpcServer")
	registry.RegisterNetworkServiceRegistryServer(w.grpcServer, w.registryServer)
	logrus.Infof("Registering NetworkServiceServer with grpcServer")
	networkservice.RegisterNetworkServiceServer(w.grpcServer, w.networkServiceServer)
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

func (w *Workspace) Directory() string {
	return w.directory
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
		if w.directory != "" {
			os.RemoveAll(w.directory)
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
