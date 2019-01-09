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
	remotePeers map[string]*remotePeerDescriptor
	dataplanes  map[string]context.CancelFunc
	model.ModelListenerImpl
}

type remotePeerDescriptor struct {
	xconCounter int
	cancel      context.CancelFunc
}

func NewMonitorCrossConnectClient(monitor monitor_crossconnect_server.MonitorCrossConnectServer, model model.Model) *NsmMonitorCrossConnectClient {
	rv := &NsmMonitorCrossConnectClient{
		monitor:     monitor,
		model:       model,
		remotePeers: map[string]*remotePeerDescriptor{},
		dataplanes:  map[string]context.CancelFunc{},
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
	ctx, cancel := context.WithCancel(context.Background())
	client.dataplanes[dataplane.RegisteredName] = cancel
	go client.dataplaneCrossConnectMonitor(dataplane, ctx)
}

func (client *NsmMonitorCrossConnectClient) DataplaneDeleted(dataplane *model.Dataplane) {
	client.dataplanes[dataplane.RegisteredName]()
	delete(client.dataplanes, dataplane.RegisteredName)
}

func (client *NsmMonitorCrossConnectClient) ClientConnectionAdded(clientConnection *model.ClientConnection) {
	if clientConnection.RemoteNsm == nil {
		return
	}

	if remotePeer, exist := client.remotePeers[clientConnection.RemoteNsm.Name]; exist {
		remotePeer.xconCounter++
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	client.remotePeers[clientConnection.RemoteNsm.Name] = &remotePeerDescriptor{cancel: cancel}
	go client.remotePeerCrossConnectMonitor(clientConnection.RemoteNsm, ctx)
}

func (client *NsmMonitorCrossConnectClient) ClientConnectionDeleted(clientConnection *model.ClientConnection) {
	if clientConnection.RemoteNsm == nil {
		return
	}
	remotePeer := client.remotePeers[clientConnection.RemoteNsm.Name]
	remotePeer.xconCounter--
	if remotePeer.xconCounter == 0 {
		remotePeer.cancel()
		delete(client.remotePeers, clientConnection.RemoteNsm.Name)
	}
}

// dataplaneCrossConnectMonitor is per registered dataplane crossconnect monitoring routine.
// It creates a grpc client for the socket advertsied by the dataplane and listens for a stream of Cross Connect Events.
// If it detects a failure of the connection, it will indicate that dataplane is no longer operational. In this case
// monitor will remove all dataplane connections and will terminate itself.
func (client *NsmMonitorCrossConnectClient) dataplaneCrossConnectMonitor(dataplane *model.Dataplane, ctx context.Context) {
	logrus.Infof("Connecting to Dataplane %s %s", dataplane.RegisteredName, dataplane.SocketLocation)
	conn, err := dial(context.Background(), "unix", dataplane.SocketLocation)
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", dataplane.SocketLocation, err)
		return
	}
	defer conn.Close()

	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	stream, err := monitorClient.MonitorCrossConnects(ctx, &empty.Empty{})
	if err != nil {
		logrus.Error(err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Context timeout exceeded...")
			return
		default:
			logrus.Info("Recv CrossConnect...")
			event, err := stream.Recv()
			if err != nil {
				logrus.Error(err)
				//TODO (lobkovilya): handle dataplane dies
				return
			}
			logrus.Infof("Receive event: %s %s", event.Type, event.CrossConnects)

			for _, xcon := range event.GetCrossConnects() {
				if event.GetType() == crossconnect.CrossConnectEventType_UPDATE {
					clientConnection := client.model.GetClientConnectionByXcon(xcon.Id)
					if clientConnection != nil {
						clientConnection.Xcon = xcon
						client.model.UpdateClientConnection(clientConnection)
					}

					client.monitor.UpdateCrossConnect(xcon)
				}
				if event.GetType() == crossconnect.CrossConnectEventType_DELETE {
					clientConnection := client.model.GetClientConnectionByXcon(xcon.Id)
					if clientConnection != nil {
						client.model.DeleteClientConnection(clientConnection.ConnectionId)
					}

					client.monitor.DeleteCrossConnect(xcon)
				}
				if event.GetType() == crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER {
					client.monitor.UpdateCrossConnect(xcon)
					//TODO (lobkovilya): reconciling
				}
			}
		}
	}
}

//remotePeerCrossConnectMonitor is per registered NSM monitoring routine.
//It listens event from remote NSM (remote peers) and updates state of xcon if ones is down
func (client *NsmMonitorCrossConnectClient) remotePeerCrossConnectMonitor(remotePeer *registry.NetworkServiceManager, ctx context.Context) {
	logrus.Infof("Connecting to Remote NSM: %s", remotePeer.Name)
	conn, err := grpc.Dial(remotePeer.Url, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("Failed to dial Network Service Registry %s at %s: %s", remotePeer.GetName(), remotePeer.Url, err)
		return
	}
	defer conn.Close()

	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	stream, err := monitorClient.MonitorCrossConnects(ctx, &empty.Empty{})
	if err != nil {
		logrus.Error(err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Context timeout exceeded...")
			return
		default:
			logrus.Info("Recv CrossConnect...")
			event, err := stream.Recv()
			if err != nil {
				logrus.Error(err)
				//TODO (lobkovilya): handle remote NSM dies
				return
			}
			logrus.Infof("Receive event: %s %s", event.Type, event.CrossConnects)

			if event.GetType() == crossconnect.CrossConnectEventType_UPDATE {
				for _, xcon := range event.GetCrossConnects() {
					if xcon.State == crossconnect.CrossConnectState_DST_DOWN {
						clientConnection := client.model.GetClientConnectionByDst(xcon.GetRemoteSource().GetId())
						if clientConnection != nil {
							clientConnection.Xcon.State = xcon.State
							client.model.UpdateClientConnection(clientConnection)
						}
					}
				}
			}

		}
	}
}
