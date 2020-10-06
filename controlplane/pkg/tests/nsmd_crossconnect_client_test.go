// Copyright (c) 2018-2020 Cisco Systems, Inc and/or its affiliates.
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
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	connectionMonitor "github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"
)

func TestCCServerEmpty(t *testing.T) {
	g := NewWithT(t)

	myModel := model.NewModel()

	crossConnectAddress := "127.0.0.1:0"

	grpcServer, monitor, sock, err := startAPIServer(myModel, crossConnectAddress)
	defer grpcServer.Stop()

	crossConnectAddress = sock.Addr().String()

	g.Expect(err).To(BeNil())

	monitor.Update(context.Background(), &crossconnect.CrossConnect{
		Id:      "cc1",
		Payload: "json_data",
	})
	events := readNMSDCrossConnectEvents(crossConnectAddress, 1)

	g.Expect(len(events)).To(Equal(1))

	g.Expect(events[0].CrossConnects["cc1"].Payload).To(Equal("json_data"))
}
func TestNSMDCrossConnectClientRemote(t *testing.T) {
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	defer srv.Stop()

	mon := connectionMonitor.NewMonitorServer("LocalConnection")

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
				Source: &connection.Connection{
					Id: "1",
					Path: &connection.Path{
						PathSegments: []*connection.PathSegment{
							{
								Name: "nsm1",
							},
							{
								Name: "nsm2",
							},
						},
					},
				},
				Destination: &connection.Connection{
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
	evt := msg.(*connection.ConnectionEvent)
	ents := evt.Connections
	g.Expect(len(ents)).To(Equal(1))
	conn, ok := ents["1"]
	g.Expect(ok).To(Equal(true))
	g.Expect(conn.Id).To(Equal("1"))
	g.Expect(len(conn.GetPath().GetPathSegments())).To(Equal(2))
	g.Expect(conn.GetPath().GetPathSegments()[1].GetName()).To(Equal("nsm2"))
}

func TestNSMDCrossConnectClientLocal(t *testing.T) {
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	defer srv.Stop()

	mon := connectionMonitor.NewMonitorServer("LocalConnection")

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
				Source: &connection.Connection{
					Id: "1",
					Path: &connection.Path{
						PathSegments: []*connection.PathSegment{
							{
								Name: "nsm1",
							},
						},
					},
				},
				Destination: &connection.Connection{
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
	evt := msg.(*connection.ConnectionEvent)
	ents := evt.Connections
	g.Expect(len(ents)).To(Equal(1))
	conn, ok := ents["1"]
	g.Expect(ok).To(Equal(true))
	g.Expect(conn.Id).To(Equal("1"))
}
