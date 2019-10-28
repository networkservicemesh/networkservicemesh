// Copyright (c) 2019 Cisco Systems, Inc and/or its affiliates.
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

package tests

import (
	"context"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/local"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/remote"
)

type eventCollector struct {
	messages []interface{}
	events   chan interface{}
}

func (e *eventCollector) SendMsg(msg interface{}) error {
	e.messages = append(e.messages, msg)
	e.events <- msg
	return nil
}

func TestNSMDCrossConnectClientRemote(t *testing.T) {
	_ = os.Setenv(tools.InsecureEnv, "true")
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	defer srv.Stop()

	mon := remote.NewMonitorServer()

	msgs := &eventCollector{
		messages: []interface{}{},
		events:   make(chan interface{}),
	}
	mon.AddRecipient(msgs)

	<-msgs.events // Read initial

	srv.monitorCrossConnectClient.ClientConnectionUpdated(context.Background(),
		&model.ClientConnection{
			ConnectionID: "1",
			Xcon:         nil,
		},
		&model.ClientConnection{
			ConnectionID: "1",
			Xcon: &crossconnect.CrossConnect{
				Source: &unified.Connection{
					Id:                     "1",
					NetworkServiceManagers: []string{"nsm1", "nsm2"},
				},
				Destination: &unified.Connection{
					Id: "2",
				},
			},
			Monitor: mon,
		},
	)

	var msg interface{}
	select {
	case msg = <-msgs.events:
		break
	case <-time.After(200 * time.Millisecond):
		g.Expect(true).To(BeNil(), "Timeout")
	}
	g.Expect(len(msgs.messages)).To(Equal(2))
	evt := msg.(*remote_connection.ConnectionEvent)
	ents := evt.Connections
	g.Expect(len(ents)).To(Equal(1))
	conn, ok := ents["1"]
	g.Expect(ok).To(Equal(true))
	g.Expect(conn.Id).To(Equal("1"))
	g.Expect(conn.DestinationNetworkServiceManagerName).To(Equal("nsm2"))
}

func TestNSMDCrossConnectClientLocal(t *testing.T) {
	_ = os.Setenv(tools.InsecureEnv, "true")
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	defer srv.Stop()

	mon := local.NewMonitorServer()

	msgs := &eventCollector{
		messages: []interface{}{},
		events:   make(chan interface{}),
	}
	mon.AddRecipient(msgs)

	<-msgs.events // Read initial

	srv.monitorCrossConnectClient.ClientConnectionUpdated(context.Background(),
		&model.ClientConnection{
			ConnectionID: "1",
			Xcon:         nil,
		},
		&model.ClientConnection{
			ConnectionID: "1",
			Xcon: &crossconnect.CrossConnect{
				Source: &unified.Connection{
					Id:                     "1",
					NetworkServiceManagers: []string{"nsm1"},
				},
				Destination: &unified.Connection{
					Id: "2",
				},
			},
			Monitor: mon,
		},
	)

	var msg interface{}
	select {
	case msg = <-msgs.events:
		break
	case <-time.After(200 * time.Millisecond):
		g.Expect(true).To(BeNil(), "Timeout")
	}
	g.Expect(len(msgs.messages)).To(Equal(2))
	evt := msg.(*local_connection.ConnectionEvent)
	ents := evt.Connections
	g.Expect(len(ents)).To(Equal(1))
	conn, ok := ents["1"]
	g.Expect(ok).To(Equal(true))
	g.Expect(conn.Id).To(Equal("1"))
}
