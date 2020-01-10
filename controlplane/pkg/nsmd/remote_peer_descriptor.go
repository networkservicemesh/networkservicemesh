// Copyright (c) 2020 Cisco Systems, Inc.
//
// SPDX-License-Identifier: Apache-2.0
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
	"sync"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

//RemotePeerDescriptor represents network service manager remote peer
type RemotePeerDescriptor interface {
	sync.Locker
	AddConnection(connection *model.ClientConnection)
	RemoveConnection(connection *model.ClientConnection)
	Cancel()
	Context() context.Context
	Reset()
	IsCanceled() bool
	HasConnection() bool
	RemoteNsm() *registry.NetworkServiceManager
}

type remotePeerDescriptor struct {
	sync.Mutex
	connections map[string]*model.ClientConnection
	cancel      context.CancelFunc
	canceled    bool
	remoteNsm   *registry.NetworkServiceManager
	context     context.Context
}

func (r *remotePeerDescriptor) HasConnection() bool {
	return len(r.connections) > 0
}

func (r *remotePeerDescriptor) Context() context.Context {
	return r.context
}

func (r *remotePeerDescriptor) AddConnection(connection *model.ClientConnection) {
	r.connections[connection.ConnectionID] = connection
}

func (r *remotePeerDescriptor) RemoveConnection(connection *model.ClientConnection) {
	delete(r.connections, connection.ConnectionID)
}

func (r *remotePeerDescriptor) Cancel() {
	if r.cancel != nil {
		r.cancel()
	}
	r.canceled = true
}

func (r *remotePeerDescriptor) Reset() {
	r.context, r.cancel = context.WithCancel(context.Background())
	r.canceled = false
}

func (r *remotePeerDescriptor) IsCanceled() bool {
	return r.canceled
}

func (r *remotePeerDescriptor) RemoteNsm() *registry.NetworkServiceManager {
	return r.remoteNsm
}

//NewRemotePeerDescriptor represents network service manager remote peer
func NewRemotePeerDescriptor(conn *model.ClientConnection) RemotePeerDescriptor {
	result := &remotePeerDescriptor{
		remoteNsm: conn.RemoteNsm,
		connections: map[string]*model.ClientConnection{
			conn.ConnectionID: conn,
		},
	}
	result.Reset()
	return result
}
