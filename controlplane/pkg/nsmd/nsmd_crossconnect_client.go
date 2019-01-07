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
	"net"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type NsmMonitorCrossConnectClient struct {
	monitor     monitor_crossconnect_server.MonitorCrossConnectServer // All connections is here
	model       model.Model
	remotePeers map[string]*registry.NetworkServiceManager
	model.ModelListenerImpl
}

func NewMonitorCrossConnectClient(monitor monitor_crossconnect_server.MonitorCrossConnectServer, model model.Model) *NsmMonitorCrossConnectClient {
	rv := &NsmMonitorCrossConnectClient{
		monitor:     monitor,
		model:       model,
		remotePeers: map[string]*registry.NetworkServiceManager{},
	}
	return rv
}

func dial(ctx context.Context, network string, address string) (*grpc.ClientConn, error) {
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.Dial(network, addr)
		}),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer)),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	return conn, err
}

func (client *NsmMonitorCrossConnectClient) DataplaneAdded(dataplane *model.Dataplane) {
	go client.dataplaneCrossConnectMonitor(dataplane)
}

func (client *NsmMonitorCrossConnectClient) DataplaneDeleted(dataplane *model.Dataplane) {
	//TODO (lobkovilya): delete dataplane
}

func (client *NsmMonitorCrossConnectClient) ClientConnectionAdded(clientConnection *model.ClientConnection) {
	if clientConnection.RemoteNsm == nil {
		return
	}

	if _, exist := client.remotePeers[clientConnection.RemoteNsm.Name]; exist {
		return
	}

	client.remotePeers[clientConnection.RemoteNsm.Name] = clientConnection.RemoteNsm
	go client.remotePeerCrossConnectMonitor(clientConnection.RemoteNsm)
}

// dataplaneMonitor is per registered dataplane crossconnect monitoring routine.
// It creates a grpc client for the socket advertsied by the dataplane and listens for a stream of Cross Connect Events.
// If it detects a failure of the connection, it will indicate that dataplane is no longer operational. In this case
// monitor will remove all dataplane connections and will terminate itself.
func (client *NsmMonitorCrossConnectClient) dataplaneCrossConnectMonitor(dataplane *model.Dataplane) {
	logrus.Infof("Connecting to Dataplane %s %s", dataplane.RegisteredName, dataplane.SocketLocation)
	conn, err := dial(context.Background(), "unix", dataplane.SocketLocation)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", dataplane.SocketLocation, err)
		return
	}
	defer conn.Close()
	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	if err := client.monitorCrossConnects(monitorClient); err != nil {
		logrus.Error(err)
		//TODO(lobkovilya): delete dataplane
	}
}

func (client *NsmMonitorCrossConnectClient) remotePeerCrossConnectMonitor(remotePeer *registry.NetworkServiceManager) {
	logrus.Infof("Connecting to Remote NSM: %s", remotePeer.Name)
	conn, err := grpc.Dial(remotePeer.Url, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("Failed to dial Network Service Registry %s at %s: %s", remotePeer.GetName(), remotePeer.Url, err)
		return
	}
	defer conn.Close()
	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	if err := client.monitorCrossConnects(monitorClient); err != nil {
		logrus.Error(err)
		//TODO(lobkovilya): delete remote peer
	}
}

func (client *NsmMonitorCrossConnectClient) monitorCrossConnects(xconClient crossconnect.MonitorCrossConnectClient) error {
	// Looping indefinetly or until grpc returns an error indicating the other end closed connection.
	stream, err := xconClient.MonitorCrossConnects(context.Background(), &empty.Empty{})
	if err != nil {
		return err
	}
	for {
		logrus.Info("Recv CrossConnect...")
		event, err := stream.Recv()
		if err != nil {
			return err
		}
		logrus.Infof("Receive event: %s %s", event.Type, event.CrossConnects)

		for _, xcon := range event.GetCrossConnects() {
			if event.GetType() == crossconnect.CrossConnectEventType_UPDATE {
				clientConnection := client.model.GetClientConnectionByXcon(xcon.Id)
				if clientConnection != nil {
					clientConnection.Xcon = xcon
					client.model.UpdateClientConnection(clientConnection)
				}
				// Pass object
				logrus.Infof("Sending UPDATE event to monitor: %v", xcon)
				client.monitor.UpdateCrossConnect(xcon)
			}
			if event.GetType() == crossconnect.CrossConnectEventType_DELETE {
				clientConnection := client.model.GetClientConnectionByXcon(xcon.Id)
				if clientConnection != nil {
					client.model.DeleteClientConnection(clientConnection.ConnectionId)
				}

				// Pass object
				client.monitor.DeleteCrossConnect(xcon)
			}
			if event.GetType() == crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER {
				//TODO (lobkovilya): reconciling
			}
		}
	}
}
