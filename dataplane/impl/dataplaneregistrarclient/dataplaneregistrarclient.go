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

package dataplaneregistrarclient

import (
	"context"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplaneregistrar"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"
)

var (
	registrationRetryInterval = 30 * time.Second
)

type DataplaneRegistrarClient struct {
	registrationRetryInterval time.Duration
	registrarSocket           string
}

type dataplaneRegistration struct {
	registrar       *DataplaneRegistrarClient
	dataplaneName   string
	dataplaneSocket string
	cancelFunc      context.CancelFunc
	onConnect       OnConnectFunc
	onDisconnect    OnDisConnectFunc
	client          dataplaneregistrar.DataplaneRegistrationClient
}

type OnConnectFunc func() error
type OnDisConnectFunc func() error

func (dr *dataplaneRegistration) register(ctx context.Context) {
	logrus.Info("Registering with NetworkServiceManager")
	logrus.Infof("Retry interval: %s", dr.registrar.registrationRetryInterval)
	tools.WaitForPortAvailable(context.Background(), "unix", dr.registrar.registrarSocket, 100*time.Millisecond)
	ticker := time.NewTicker(dr.registrar.registrationRetryInterval)
	for ; true; <-ticker.C {
		select {
		case <-ctx.Done():
			return
		default:
			err := dr.tryRegistration(ctx)
			if err == nil {
				return
			}
		}
	}
}

func (dr *dataplaneRegistration) tryRegistration(ctx context.Context) error {
	logrus.Infof("Trying to register %s on socket %s", dr.dataplaneName, dr.registrar.registrarSocket)
	if _, err := os.Stat(dr.registrar.registrarSocket); err != nil {
		logrus.Errorf("%s: failure to access nsm socket at \"%s\" with error: %+v, exiting...", dr.dataplaneName, dr.registrar.registrarSocket, err)
		return err
	}
	conn, err := tools.SocketOperationCheck(dr.registrar.registrarSocket)
	if err != nil {
		logrus.Errorf("%s: failure to communicate with the socket \"%s\" with error: %+v", dr.dataplaneName, dr.registrar.registrarSocket, err)
		return err
	}
	logrus.Infof("%s: connection to dataplane registrar socket \"%s\" succeeded.", dr.dataplaneName, dr.registrar.registrarSocket)

	dr.client = dataplaneregistrar.NewDataplaneRegistrationClient(conn)
	req := &dataplaneregistrar.DataplaneRegistrationRequest{
		DataplaneName:   dr.dataplaneName,
		DataplaneSocket: dr.dataplaneSocket,
	}
	_, err = dr.client.RequestDataplaneRegistration(ctx, req)
	logrus.Infof("%s: send request to Dataplane Registrar: %+v", dr.dataplaneName, req)
	if err != nil {
		logrus.Infof("%s: failure to create grpc client for RequestDataplaneRegistration on socket %s", dr.dataplaneName, dr.registrar.registrarSocket)
		return err
	}
	if dr.onConnect != nil {
		dr.onConnect()
	}
	go dr.livenessMonitor(ctx)
	return nil
}

// livenessMonitor is a stream initiated by NSM to inform the dataplane that NSM is still alive and
// no re-registration is required. Detection a failure on this "channel" will mean
// that NSM is gone and the dataplane needs to start re-registration logic.
func (dr *dataplaneRegistration) livenessMonitor(ctx context.Context) {
	logrus.Infof("Starting DataplaneRegistrarClient liveliness monitor")
	stream, err := dr.client.RequestLiveness(context.Background())
	if err != nil {
		logrus.Errorf("%s: fail to create liveness grpc channel with NSM with error: %s, grpc code: %+v", dr.dataplaneName, err.Error(), status.Convert(err).Code())
		return
	}
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("DataplaneRegistrarClient cancelled, cleaning up")
			if dr.onDisconnect != nil {
				dr.onDisconnect()
			}
			return
		default:
			err := stream.RecvMsg(&empty.Empty{})
			if err != nil {
				logrus.Errorf("%s: fail to receive from liveness grpc channel with error: %s, grpc code: %+v", dr.dataplaneName, err.Error(), status.Convert(err).Code())
				if dr.onConnect != nil {
					dr.onDisconnect()
				}
				go dr.register(ctx)
				return
			}
		}
	}
}

func (dr *dataplaneRegistration) Close() {
	dr.cancelFunc()
	conn, err := tools.SocketOperationCheck(dr.registrar.registrarSocket)
	if err != nil {
		logrus.Errorf("%s: failure to communicate with the socket %s with error: %+v", dr.dataplaneName, dr.registrar.registrarSocket, err)
		return
	}
	logrus.Infof("%s: connection to dataplane registrar socket %s succeeded.", dr.dataplaneName, dr.registrar.registrarSocket)
	client := dataplaneregistrar.NewDataplaneUnRegistrationClient(conn)
	client.RequestDataplaneUnRegistration(context.Background(), &dataplaneregistrar.DataplaneUnRegistrationRequest{
		DataplaneName: dr.dataplaneName,
	})
}

func NewDataplaneRegistrarClient(registrarSocket string) *DataplaneRegistrarClient {
	return &DataplaneRegistrarClient{
		registrationRetryInterval: registrationRetryInterval,
		registrarSocket:           registrarSocket,
	}
}

func (n *DataplaneRegistrarClient) Register(ctx context.Context, dataplaneName, dataplaneSocket string, onConnect OnConnectFunc, onDisconnect OnDisConnectFunc) *dataplaneRegistration {
	ctx, cancelFunc := context.WithCancel(ctx)
	rv := &dataplaneRegistration{
		registrar:       n,
		dataplaneName:   dataplaneName,
		dataplaneSocket: dataplaneSocket,
		onConnect:       onConnect,
		onDisconnect:    onDisconnect,
		cancelFunc:      cancelFunc,
	}
	rv.register(ctx)
	return rv
}
