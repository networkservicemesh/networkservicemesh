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

	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarderregistrar"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var (
	registrationRetryInterval = 30 * time.Second
)

type ForwarderRegistrarClient struct {
	registrationRetryInterval time.Duration
	registrarSocket           net.Addr
}

// ForwarderRegistration contains Forwarder registrar client info and connection events callbacks
type ForwarderRegistration struct {
	registrar       *ForwarderRegistrarClient
	forwarderName   string
	forwarderSocket string
	cancelFunc      context.CancelFunc
	onConnect       OnConnectFunc
	onDisconnect    OnDisConnectFunc
	client          forwarderregistrar.ForwarderRegistrationClient
	wasRegistered   bool
}

type OnConnectFunc func() error
type OnDisConnectFunc func() error

func (dr *ForwarderRegistration) register(ctx context.Context) {
	logrus.Info("Registering with NetworkServiceManager")
	logrus.Infof("Retry interval: %s", dr.registrar.registrationRetryInterval)

	// Wait fo NSMD to be ready to register forwarder.
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

func (dr *ForwarderRegistration) tryRegistration(ctx context.Context) error {
	logrus.Infof("Trying to register %s on socket %v", dr.forwarderName, dr.registrar.registrarSocket)
	if dr.registrar.registrarSocket.Network() == "unix" {
		if _, err := os.Stat(dr.registrar.registrarSocket.String()); err != nil {
			logrus.Errorf("%s: failure to access nsm socket at \"%v\" with error: %+v, exiting...", dr.forwarderName, dr.registrar.registrarSocket, err)
			return err
		}
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := tools.DialContext(dialCtx, dr.registrar.registrarSocket)
	if err != nil {
		logrus.Errorf("%s: failure to communicate with the socket \"%v\" with error: %+v", dr.forwarderName, dr.registrar.registrarSocket, err)
		return err
	}
	logrus.Infof("%s: connection to forwarder registrar socket \"%v\" succeeded.", dr.forwarderName, dr.registrar.registrarSocket)

	dr.client = forwarderregistrar.NewForwarderRegistrationClient(conn)
	req := &forwarderregistrar.ForwarderRegistrationRequest{
		ForwarderName:   dr.forwarderName,
		ForwarderSocket: dr.forwarderSocket,
	}
	_, err = dr.client.RequestForwarderRegistration(ctx, req)
	logrus.Infof("%s: send request to Forwarder Registrar: %+v", dr.forwarderName, req)
	if err != nil {
		logrus.Infof("%s: failure to create grpc client for RequestForwarderRegistration on socket %v", dr.forwarderName, dr.registrar.registrarSocket)
		return err
	}
	if dr.onConnect != nil {
		if connectErr := dr.onConnect(); connectErr != nil {
			// TODO determine if we should clean up and exit here
			logrus.Error(connectErr)
		}
		dr.wasRegistered = true
	}
	go dr.livenessMonitor(ctx)
	return nil
}

// livenessMonitor is a stream initiated by NSM to inform the forwarder that NSM is still alive and
// no re-registration is required. Detection a failure on this "channel" will mean
// that NSM is gone and the forwarder needs to start re-registration logic.
func (dr *ForwarderRegistration) livenessMonitor(ctx context.Context) {
	logrus.Infof("Starting ForwarderRegistrarClient liveliness monitor")
	stream, err := dr.client.RequestLiveness(context.Background())
	if err != nil {
		logrus.Errorf("%s: fail to create liveness grpc channel with NSM with error: %s, grpc code: %+v", dr.forwarderName, err.Error(), status.Convert(err).Code())
		return
	}
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("ForwarderRegistrarClient cancelled, cleaning up")
			if dr.onDisconnect != nil {
				if disconnectErr := dr.onDisconnect(); disconnectErr != nil {
					logrus.Error(disconnectErr)
				}
				dr.wasRegistered = false
			}
			return
		default:
			err := stream.RecvMsg(&empty.Empty{})
			if err != nil {
				logrus.Errorf("%s: fail to receive from liveness grpc channel with error: %s, grpc code: %+v", dr.forwarderName, err.Error(), status.Convert(err).Code())
				if dr.onConnect != nil {
					if disconnectErr := dr.onDisconnect(); disconnectErr != nil {
						logrus.Error(disconnectErr)
					}
					dr.wasRegistered = false
				}
				go dr.register(ctx) // Use base ctx, to not go into deep
				return
			}
		}
	}
}

// Close forwarder registrar client
func (dr *ForwarderRegistration) Close() {
	dr.cancelFunc()

	if dr.wasRegistered {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		conn, err := tools.DialContext(ctx, dr.registrar.registrarSocket)
		if err != nil {
			logrus.Errorf("%s: failure to communicate with the socket %v with error: %+v", dr.forwarderName, dr.registrar.registrarSocket, err)
			return
		}
		logrus.Infof("%s: connection to forwarder registrar socket %v succeeded.", dr.forwarderName, dr.registrar.registrarSocket)
		client := forwarderregistrar.NewForwarderUnRegistrationClient(conn)

		unregCtx, unRegCancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer unRegCancel()

		_, _ = client.RequestForwarderUnRegistration(unregCtx, &forwarderregistrar.ForwarderUnRegistrationRequest{
			ForwarderName: dr.forwarderName,
		})
	}
}

func NewForwarderRegistrarClient(network, registrarSocket string) *ForwarderRegistrarClient {
	return &ForwarderRegistrarClient{
		registrationRetryInterval: registrationRetryInterval,
		registrarSocket: &net.UnixAddr{
			Name: registrarSocket,
			Net:  network,
		},
	}
}

// Register creates and register new ForwarderRegistration client
func (n *ForwarderRegistrarClient) Register(ctx context.Context, forwarderName, forwarderSocket string, onConnect OnConnectFunc, onDisconnect OnDisConnectFunc) *ForwarderRegistration {
	ctx, cancelFunc := context.WithCancel(ctx)
	rv := &ForwarderRegistration{
		registrar:       n,
		forwarderName:   forwarderName,
		forwarderSocket: forwarderSocket,
		onConnect:       onConnect,
		onDisconnect:    onDisconnect,
		cancelFunc:      cancelFunc,
	}
	rv.register(ctx)
	return rv
}
