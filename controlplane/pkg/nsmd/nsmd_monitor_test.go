// Copyright (c) 2019 Cisco Systems, Inc.
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
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const sendPeriod = time.Millisecond * 15
const testingTime = time.Second / 4

func TestNsmdMonitorShouldHandleServerShutdown(t *testing.T) {
	g := gomega.NewWithT(t)
	port, stop := setupServer(g)
	go func() {
		<-time.After(testingTime / 2)
		stop()
	}()
	err := startClient(g, testingTime, port)
	g.Expect(err).ShouldNot(gomega.BeNil())
}

func startClient(g *gomega.WithT, timeout time.Duration, serverPort int) error {
	conn, err := tools.DialTCP(fmt.Sprintf(":%v", serverPort))
	g.Expect(err).Should(gomega.BeNil())
	client, err := connectionmonitor.NewMonitorClient(conn, &connection.MonitorScopeSelector{})
	g.Expect(err).Should(gomega.BeNil())
	for {
		select {
		case <-time.After(timeout):
			return nil
		case result := <-client.ErrorChannel():
			return result
		case event := <-client.EventChannel():
			if event != nil {
				logrus.Info(event.Message())
			}
		}
	}
}

func setupServer(g *gomega.WithT) (int, func()) {
	grpcServer := tools.NewServer(context.Background())
	remoteMonitor := remote.NewMonitorServer(nil)
	connection.RegisterMonitorConnectionServer(grpcServer, remoteMonitor)
	l, err := net.Listen("tcp", ":0")
	g.Expect(err).Should(gomega.BeNil())
	go func() {
		grpcServer.Serve(l)
	}()
	go func() {
		for {
			<-time.After(sendPeriod)
			remoteMonitor.SendAll(&testEvent{})
		}
	}()

	return l.Addr().(*net.TCPAddr).Port,
		grpcServer.Stop
}

type testEvent struct {
}

func (*testEvent) EventType() monitor.EventType {
	return monitor.EventTypeUpdate
}

func (*testEvent) Entities() map[string]monitor.Entity {
	return nil
}

func (t *testEvent) Message() (interface{}, error) {
	return &connection.ConnectionEvent{
		Type:        connection.ConnectionEventType_UPDATE,
		Connections: nil,
	}, nil
}

func (*testEvent) Context() context.Context {
	return context.Background()
}
