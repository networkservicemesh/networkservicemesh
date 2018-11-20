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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"time"
)

type NSMMonitorCrossConnectClient interface {
	Register(model model.Model)
	Unregister(model model.Model)
}

type nsmMonitorCrossConnectClient struct {
	monitor    monitor_crossconnect_server.MonitorCrossConnectServer // All connections is here
	model      model.Model
	dataplanes map[string]*dataplaneCrossConnectInfo
}

func (client *nsmMonitorCrossConnectClient) Register(model model.Model) {
	model.AddListener(client)
}

func (client *nsmMonitorCrossConnectClient) Unregister(model model.Model) {
	model.RemoveListener(client)
}

type dataplaneCrossConnectInfo struct {
	crossConnects map[string]*crossconnect.CrossConnect
}

func (client *nsmMonitorCrossConnectClient) EndpointAdded(endpoint *registry.NetworkServiceEndpoint) {
}

func (client *nsmMonitorCrossConnectClient) EndpointDeleted(endpoint *registry.NetworkServiceEndpoint) {
}

func (client *nsmMonitorCrossConnectClient) DataplaneAdded(dataplane *model.Dataplane) {
	info := &dataplaneCrossConnectInfo{
		crossConnects: make(map[string]*crossconnect.CrossConnect),
	}
	client.dataplanes[dataplane.RegisteredName] = info
	go client.dataplaneCrossConnectMonitor(dataplane, info)
}

func (client *nsmMonitorCrossConnectClient) DataplaneDeleted(dataplane *model.Dataplane) {
}

func NewMonitorCrossConnectClient(monitor monitor_crossconnect_server.MonitorCrossConnectServer) NSMMonitorCrossConnectClient {
	rv := &nsmMonitorCrossConnectClient{
		dataplanes: make(map[string]*dataplaneCrossConnectInfo),
		monitor:    monitor,
	}
	return rv
}

func dial(ctx context.Context, network string, address string) (*grpc.ClientConn, error) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.Dial(network, addr)
		}),
	)
	return conn, err
}

// dataplaneMonitor is per registered dataplane crossconnect monitoring routine.
// It creates a grpc client for the socket advertsied by the dataplane and listens for a stream of Cross Connect Events.
// If it detects a failure of the connection, it will indicate that dataplane is no longer operational. In this case
// monitor will remove all dataplane connections and will terminate itself.
func (client *nsmMonitorCrossConnectClient) dataplaneCrossConnectMonitor(dataplane *model.Dataplane, dataplaneInfo *dataplaneCrossConnectInfo) {
	var err error
	if dataplane == nil {
		logrus.Errorf("Dataplane object store does not have registered plugin %s", dataplane.RegisteredName)
		return
	}
	logrus.Infof("Connecting to Dataplane %s %s", dataplane.RegisteredName, dataplane.SocketLocation)
	conn, err := dial(context.Background(), "unix", dataplane.SocketLocation)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", dataplane.SocketLocation, err)
		return
	}
	defer conn.Close()
	dataplaneClient := crossconnect.NewMonitorCrossConnectClient(conn)

	// Looping indefinetly or until grpc returns an error indicating the other end closed connection.
	stream, err := dataplaneClient.MonitorCrossConnects(context.Background(), &empty.Empty{})
	if err != nil {
		logrus.Warningf("Fail to create update grpc channel for Dataplane CrossConnect monitor %s with error: %+v. Dataplane could not support cross connects monitoring", dataplane.RegisteredName, err)
		return
	}
	for {
		logrus.Infof("Recv from Dataplane CrossConnect %s %s", dataplane.RegisteredName, dataplane.SocketLocation)
		event, err := stream.Recv()
		if err != nil {
			logrus.Errorf("fail to receive event from grpc channel for Dataplane %s with error: %+v.", dataplane.RegisteredName, err)
			// We need to remove all connections from this dataplane.

			for _, xcon := range dataplaneInfo.crossConnects {
				client.monitor.DeleteCrossConnect(xcon)
			}
			return
		}
		logrus.Infof("Dataplane %s informed of its parameters changes, applying new parameters %+v", dataplane.RegisteredName, event.CrossConnects)

		for _, xcon := range event.GetCrossConnects() {
			if event.GetType() == crossconnect.CrossConnectEventType_UPDATE {
				dataplaneInfo.crossConnects[xcon.GetId()] = xcon

				// Pass object
				client.monitor.UpdateCrossConnect(xcon)
			}
			if event.GetType() == crossconnect.CrossConnectEventType_DELETE {
				delete(dataplaneInfo.crossConnects, xcon.GetId())

				// Pass object
				client.monitor.DeleteCrossConnect(xcon)
			}
			if event.GetType() == crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER {
				dataplaneInfo.crossConnects[xcon.GetId()] = xcon

				// Pass object
				client.monitor.UpdateCrossConnect(xcon)
			}
		}
	}
}
