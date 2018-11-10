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
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
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
	name           string
	directory      string
	listener       net.Listener
	registryServer registry.NetworkServiceRegistryServer
	sync.Mutex
	state WorkspaceState
}

func NewWorkSpace(model model.Model, name string) (*Workspace, error) {
	logrus.Infof("Creating new workspace: %s", name)
	w := &Workspace{}
	w.state = NEW
	w.name = name
	w.directory = rootDir + w.name
	logrus.Infof("Creating new directory: %s", w.directory)
	if err := os.MkdirAll(w.directory, folderMask); err != nil {
		logrus.Errorf("can't create folder: %s, error: %v", w.directory, err)
		w.Close()
		return nil, err
	}
	socket := w.directory + "/" + WorkspaceClientSocket
	logrus.Infof("Creating new listener on: %s", socket)
	listener, err := NewCustomListener(socket)
	if err != nil {
		logrus.Error(err)
		w.Close()
		return nil, err
	}
	w.listener = listener
	logrus.Infof("Creating new RegistryServer")
	w.registryServer = NewRegistryServer(model)

	logrus.Infof("Creating new GRPC Server")
	grpcServer := grpc.NewServer()
	logrus.Infof("Registering registryServer with grpcServer")
	registry.RegisterNetworkServiceRegistryServer(grpcServer, w.registryServer)

	go func() {
		err = grpcServer.Serve(w.listener)
		if err != nil {
			logrus.Error(err)
			w.Close()
			return
		}
		w.state = RUNNING
	}()
	logrus.Infof("Created new workspace: %+v", w)
	return w, nil
}

func (w *Workspace) Name() string {
	return w.name
}

func (w *Workspace) Directory() string {
	return w.directory
}

func (w *Workspace) Close() error {
	// TODO handle cleanup here on failure in NewWorkspace creation
	w.Lock()
	defer w.Unlock()
	// w.registryServer.Close()
	if w.state != CLOSED {
		err := w.listener.Close()
		if err != nil {
			return err
		}
		err = os.RemoveAll(w.directory)
		return err
	}
	return nil
}
