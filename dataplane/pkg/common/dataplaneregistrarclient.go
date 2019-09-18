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

package common

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/dataplane/api/dataplaneregistrar"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var (
	registrationRetryInterval = 30 * time.Second
)

type DataplaneRegistrarClient struct {
	registrationRetryInterval time.Duration
	registrarSocket           net.Addr
}

// DataplaneRegistration contains Dataplane registrar client info and connection events callbacks
type DataplaneRegistration struct {
	registrar       *DataplaneRegistrarClient
	dataplaneName   string
	dataplaneSocket string
	cancelFunc      context.CancelFunc
	onConnect       OnConnectFunc
	onDisconnect    OnDisConnectFunc
	client          dataplaneregistrar.DataplaneRegistrationClient
	wasRegistered   bool
}

type OnConnectFunc func() error
type OnDisConnectFunc func() error

func (dr *DataplaneRegistration) register(ctx context.Context) {
	logrus.Info("Registering with NetworkServiceManager")
	logrus.Infof("Retry interval: %s", dr.registrar.registrationRetryInterval)

	// Wait fo NSMD to be ready to register dataplane.
	_ = tools.WaitForPortAvailable(context.Background(), dr.registrar.registrarSocket.Network(), dr.registrar.registrarSocket.String(), 100*time.Millisecond)
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

func (dr *DataplaneRegistration) tryRegistration(ctx context.Context) error {
	logrus.Infof("Trying to register %s on socket %v", dr.dataplaneName, dr.registrar.registrarSocket)
	if dr.registrar.registrarSocket.Network() == "unix" {
		if _, err := os.Stat(dr.registrar.registrarSocket.String()); err != nil {
			logrus.Errorf("%s: failure to access nsm socket at \"%v\" with error: %+v, exiting...", dr.dataplaneName, dr.registrar.registrarSocket, err)
			return err
		}
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := tools.DialContext(dialCtx, dr.registrar.registrarSocket)
	if err != nil {
		logrus.Errorf("%s: failure to communicate with the socket \"%v\" with error: %+v", dr.dataplaneName, dr.registrar.registrarSocket, err)
		return err
	}
	logrus.Infof("%s: connection to dataplane registrar socket \"%v\" succeeded.", dr.dataplaneName, dr.registrar.registrarSocket)

	dr.client = dataplaneregistrar.NewDataplaneRegistrationClient(conn)
	req := &dataplaneregistrar.DataplaneRegistrationRequest{
		DataplaneName:   dr.dataplaneName,
		DataplaneSocket: dr.dataplaneSocket,
	}
	_, err = dr.client.RequestDataplaneRegistration(ctx, req)
	logrus.Infof("%s: send request to Dataplane Registrar: %+v", dr.dataplaneName, req)
	if err != nil {
		logrus.Infof("%s: failure to create grpc client for RequestDataplaneRegistration on socket %v", dr.dataplaneName, dr.registrar.registrarSocket)
		return err
	}
	if dr.onConnect != nil {
		dr.onConnect()
		dr.wasRegistered = true
	}
	go dr.livenessMonitor(ctx)
	return nil
}

// livenessMonitor is a stream initiated by NSM to inform the dataplane that NSM is still alive and
// no re-registration is required. Detection a failure on this "channel" will mean
// that NSM is gone and the dataplane needs to start re-registration logic.
func (dr *DataplaneRegistration) livenessMonitor(ctx context.Context) {
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
				dr.wasRegistered = false
			}
			return
		default:
			err := stream.RecvMsg(&empty.Empty{})
			if err != nil {
				logrus.Errorf("%s: fail to receive from liveness grpc channel with error: %s, grpc code: %+v", dr.dataplaneName, err.Error(), status.Convert(err).Code())
				if dr.onConnect != nil {
					dr.onDisconnect()
					dr.wasRegistered = false
				}
				go dr.register(ctx)
				return
			}
		}
	}
}

// Close dataplane registrar client
func (dr *DataplaneRegistration) Close() {
	dr.cancelFunc()

	if dr.wasRegistered {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		conn, err := tools.DialContext(ctx, dr.registrar.registrarSocket)
		if err != nil {
			logrus.Errorf("%s: failure to communicate with the socket %v with error: %+v", dr.dataplaneName, dr.registrar.registrarSocket, err)
			return
		}
		logrus.Infof("%s: connection to dataplane registrar socket %v succeeded.", dr.dataplaneName, dr.registrar.registrarSocket)
		client := dataplaneregistrar.NewDataplaneUnRegistrationClient(conn)

		unregCtx, unRegCancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer unRegCancel()

		_, _ = client.RequestDataplaneUnRegistration(unregCtx, &dataplaneregistrar.DataplaneUnRegistrationRequest{
			DataplaneName: dr.dataplaneName,
		})
	}
}

func NewDataplaneRegistrarClient(network, registrarSocket string) *DataplaneRegistrarClient {
	return &DataplaneRegistrarClient{
		registrationRetryInterval: registrationRetryInterval,
		registrarSocket: &net.UnixAddr{
			Name: registrarSocket,
			Net:  network,
		},
	}
}

// Register creates and register new DataplaneRegistration client
func (n *DataplaneRegistrarClient) Register(ctx context.Context, dataplaneName, dataplaneSocket string, onConnect OnConnectFunc, onDisconnect OnDisConnectFunc) *DataplaneRegistration {
	ctx, cancelFunc := context.WithCancel(ctx)
	rv := &DataplaneRegistration{
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
