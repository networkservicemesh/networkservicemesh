// Copyright (c) 2019 Cisco and/or its affiliates.
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

package nsmmonitor

import (
	"context"
	"fmt"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	nsminit "github.com/networkservicemesh/networkservicemesh/side-cars/pkg/nsm-init"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

const (
	nsmMonitorLogFormat          = "NSM Monitor: %v"
	nsmMonitorLogWithParamFormat = "NSM Monitor: %v: %v"

	nsmMonitorRetryDelay = 5 * time.Second
)

// Handler - handler to perform configuration of monitoring app
type Handler interface {
	//Connected occurs when the nsm-monitor connected
	Connected(map[string]*networkservice.Connection)
	//Healing occurs when the healing started
	Healing(conn *networkservice.Connection)
	//Closed occurs when the connection closed
	Closed(conn *networkservice.Connection)
	//ProcessHealing occurs when the restore failed, the error pass as the second parameter
	ProcessHealing(newConn *networkservice.Connection, e error)
	//Updated triggers when existing connection updated
	Updated(old, new *networkservice.Connection)
}

// App - application to perform monitoring.
type App interface {
	nsminit.NSMApp
	// SetHandler - sets a handler instance
	SetHandler(helper Handler)
	Stop()
}

//EmptyNSMMonitorHandler has empty implementation of each method of interface Handler
type EmptyNSMMonitorHandler struct {
}

//Connected occurs when the nsm-monitor connected
func (h *EmptyNSMMonitorHandler) Connected(map[string]*networkservice.Connection) {}

//Healing occurs when the healing started
func (h *EmptyNSMMonitorHandler) Healing(conn *networkservice.Connection) {}

//Closed occurs when the connection closed
func (h *EmptyNSMMonitorHandler) Closed(conn *networkservice.Connection) {}

//ProcessHealing occurs when the restore failed, the error pass as the second parameter
func (h *EmptyNSMMonitorHandler) ProcessHealing(newConn *networkservice.Connection, e error) {}

type nsmMonitorApp struct {
	connections map[string]*networkservice.Connection
	helper      Handler
	cancelFunc  context.CancelFunc

	initRecieved  bool
	recovery      bool
	configuration *common.NSConfiguration
}

func (c *nsmMonitorApp) Stop() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}

func (c *nsmMonitorApp) SetHandler(listener Handler) {
	c.helper = listener
}

func (c *nsmMonitorApp) Run() {
	// Capture signals to cleanup before exiting
	closer := jaeger.InitJaeger("nsm-monitor")
	defer func() { _ = closer.Close() }()

	go c.beginMonitoring()
}

// NewNSMMonitorApp - creates a monitoring application.
func NewNSMMonitorApp(configuration *common.NSConfiguration) App {
	return &nsmMonitorApp{
		connections:   map[string]*networkservice.Connection{},
		configuration: configuration,
	}
}

func (c *nsmMonitorApp) beginMonitoring() {
	for {
		nsmClient, err := client.NewNSMClient(context.Background(), c.configuration)
		if err != nil {
			logrus.Errorf(nsmMonitorLogWithParamFormat, "unable to create the NSM client", err)

			c.waitRetry()
			continue
		}

		logrus.Infof(nsmMonitorLogFormat, "connection to NSM established")

		//		monitorClient, err := local.NewMonitorClient(nsmClient.NsmConnection.GrpcClient)
		ctx, cancelFunc := context.WithCancel(context.Background())
		monitorClient, err := networkservice.NewMonitorConnectionClient(nsmClient.NsmConnection.GrpcClient).MonitorConnections(ctx, &networkservice.MonitorScopeSelector{})
		if err != nil {
			logrus.Errorf(nsmMonitorLogWithParamFormat, "failed to start monitor client", err)

			c.waitRetry()
			if err := nsmClient.Destroy(context.Background()); err != nil {
				logrus.Errorf("failed to close NSM client connection")
			}
			continue
		}
		c.cancelFunc = cancelFunc
		defer cancelFunc()

		for {
			if c.initRecieved && !c.recovery {
				// Performing recovery if required.
				if c.helper != nil {
					c.helper.Connected(c.connections)
				}
				// Since NSMD will setup public socket only when all connections will be ok, we need to perform request only on ones it loose.
				if c.performRecovery(nsmClient) {
					// since we not recovered, we will continue after delay
					c.waitRetry()
					continue
				} else {
					c.recovery = true
				}
			}
			if !c.readEvents(monitorClient) {
				break // If someting happened we need to retry
			}
		}

		// Close current NSM client connection.
		if err := nsmClient.Destroy(context.Background()); err != nil {
			logrus.Errorf("failed to close NSM client connection")
		}
	}
}

func (c *nsmMonitorApp) readEvents(monitorClient networkservice.MonitorConnection_MonitorConnectionsClient) bool {
	event, err := monitorClient.Recv()
	if err != nil {
		logrus.Errorf(nsmMonitorLogWithParamFormat, "NSM die, re-connecting", err)
		for _, c := range c.connections {
			c.State = networkservice.State_DOWN // Mark all as down.
		}
		return false
	}
	if event.Type == networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER {
		logrus.Infof(nsmMonitorLogFormat, "Monitor started")
		c.initRecieved = true
	}

	for _, conn := range event.GetConnections() {
		switch event.Type {
		case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, networkservice.ConnectionEventType_UPDATE:
			c.updateConnection(conn)
		case networkservice.ConnectionEventType_DELETE:
			logrus.Infof(nsmMonitorLogFormat, "Connection closed")
			if c.helper != nil {
				c.helper.Closed(conn)
			}
		}
	}
	return true
}

func (c *nsmMonitorApp) updateConnection(conn *networkservice.Connection) {
	if existingConn, exists := c.connections[conn.GetId()]; exists {
		logrus.Infof(nsmMonitorLogWithParamFormat, "Connection updated", fmt.Sprintf("%v %v", existingConn, conn))
		c.helper.Updated(existingConn, conn)
	} else {
		logrus.Infof(nsmMonitorLogWithParamFormat, "Initial connection accepted", conn)
	}
	c.connections[conn.GetId()] = conn
}

func (c *nsmMonitorApp) waitRetry() {
	logrus.Errorf(nsmMonitorLogWithParamFormat, "Retry delay", nsmMonitorRetryDelay)
	<-time.After(nsmMonitorRetryDelay)
}

func (c *nsmMonitorApp) performRecovery(nsmClient *client.NsmClient) bool {
	logrus.Infof(nsmMonitorLogFormat, "Performing recovery if needed...")

	needRetry := false
	for _, conn := range c.connections {
		if conn.State == networkservice.State_UP {
			continue
		}
		cClone := conn.Clone()

		ipCtx := cClone.Context.IpContext
		if ipCtx != nil {
			if ipCtx.DstIpAddr != "" {
				ipCtx.DstIpRequired = true
			}
			if ipCtx.SrcIpAddr != "" {
				ipCtx.SrcIpRequired = true
			}
		}

		outgoingRequest := networkservice.NetworkServiceRequest{
			Connection: cClone,
			MechanismPreferences: []*networkservice.Mechanism{
				conn.Mechanism,
			},
		}
		if c.helper != nil {
			c.helper.Healing(cClone)
		}

		outgoingConnection, err := nsmClient.NsClient.Request(context.Background(), &outgoingRequest)

		if err != nil {
			logrus.Errorf(nsmMonitorLogWithParamFormat, "failed to restore connection. Will retry", err)
			// Let's drop connection id, since we failed one time.
			conn.Id = "-"
			needRetry = true
			continue
		} else {
			logrus.Errorf(nsmMonitorLogWithParamFormat, "connection restored", outgoingConnection)
			delete(c.connections, conn.Id)
			c.connections[outgoingConnection.Id] = outgoingConnection
		}
		if c.helper != nil {
			c.helper.ProcessHealing(outgoingConnection, err)
		}
	}
	return needRetry
}
