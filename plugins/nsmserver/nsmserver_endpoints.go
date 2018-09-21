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

package nsmserver

import (
	"fmt"
	"net"
	"path"

	nsmapi "github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	nsmclient "github.com/ligato/networkservicemesh/pkg/client/clientset/versioned"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nseconnect"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// EndpointSocketBaseDir defines the location of NSM Endpoints listen socket
	EndpointSocketBaseDir = "/var/lib/networkservicemesh"
	// EndpointSocket defines the name of NSM Endpoints operations socket
	EndpointSocket = "nsm.endpoint.io.sock"
	// EndpointServiceLabel is a label which is used to select all Endpoint object
	// for a specific Network Service
	EndpointServiceLabel = "networkservicemesh.io/network-service-name"
)

type nsmEndpointServer struct {
	logger             logger.FieldLoggerPlugin
	objectStore        objectstore.Interface
	k8sClient          *kubernetes.Clientset
	nsmClient          *nsmclient.Clientset
	grpcServer         *grpc.Server
	endPointSocketPath string
	stopChannel        chan bool
	nsmNamespace       string
	nsmPodIPAddress    string
}

func (e nsmEndpointServer) AdvertiseEndpoint(ctx context.Context,
	ar *nseconnect.EndpointAdvertiseRequest) (*nseconnect.EndpointAdvertiseReply, error) {
	e.logger.Infof("Received Endpoint Advertise request: %s", ar.RequestId)

	// Compose a new Network Service Endpoint object name from Request_id which is NSE's pod
	// UUID and Network Service Name.
	endpointName := ar.RequestId + "-" + ar.NetworkEndpoint.NetworkServiceName
	// Check if there is already Network Service Endpoint object with the same name, if there is
	// success will be returned to NSE, since it is a case of NSE pod coming back up.
	_, err := e.nsmClient.NetworkserviceV1().NetworkServiceEndpoints(e.nsmNamespace).Get(endpointName, metav1.GetOptions{})
	if err == nil {
		e.logger.Warnf("Network Service Endpoint object %s already exists", endpointName)
		return &nseconnect.EndpointAdvertiseReply{
			RequestId: ar.RequestId,
			Accepted:  true,
		}, nil
	}

	if !apierrors.IsNotFound(err) {
		// something bad happened while attempting to check if the object already exists,
		// it is safer to record the error and bail out.
		e.logger.Errorf("advertise request %s fail to check if %s already exists with error: %+v", ar.RequestId, endpointName, err)
		return &nseconnect.EndpointAdvertiseReply{
			RequestId:      ar.RequestId,
			Accepted:       false,
			AdmissionError: fmt.Sprintf("advertise request %s fail to check if %s already exists with error: %+v", ar.RequestId, endpointName, err),
		}, err
	}

	endpoint := &nsmapi.NetworkServiceEndpoint{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkServiceEndpoint",
			APIVersion: "networkservicemesh.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      endpointName,
			Namespace: e.nsmNamespace,
			Labels:    map[string]string{EndpointServiceLabel: ar.NetworkEndpoint.NetworkServiceName},
		},
		Spec: ar.NetworkEndpoint,
	}
	// nsmPodIPAddress in Network Service Endpoint object will indicate that
	// it was orginated by this specific NSM and remote NSM will use this IP address
	// for NSM2NSM gRPC communication.
	endpoint.Spec.NetworkServiceHost = e.nsmPodIPAddress
	_, err = e.nsmClient.NetworkserviceV1().NetworkServiceEndpoints(e.nsmNamespace).Create(endpoint)
	if err != nil {
		// something bad happened while attempting to create a new object, logging error and exit.
		e.logger.Errorf("advertise request %s fail to create a new Network Service Endpoint object %s with error: %+v", ar.RequestId, endpointName, err)
		return &nseconnect.EndpointAdvertiseReply{
			RequestId:      ar.RequestId,
			Accepted:       false,
			AdmissionError: fmt.Sprintf("advertise request %s fail to create a new Network Service Endpoint object %s with error: %+v", ar.RequestId, endpointName, err),
		}, err
	}
	return &nseconnect.EndpointAdvertiseReply{
		RequestId: ar.RequestId,
		Accepted:  true,
	}, nil
}

func (e nsmEndpointServer) RemoveEndpoint(ctx context.Context,
	rr *nseconnect.EndpointRemoveRequest) (*nseconnect.EndpointRemoveReply, error) {
	e.logger.Infof("Received Endpoint Remove request: %+v", rr)
	return &nseconnect.EndpointRemoveReply{
		RequestId: rr.RequestId,
		Accepted:  true,
	}, nil
}

// startEndpointServer starts for a server listening for local NSEs advertise/remove
// endpoint calls
func startEndpointServer(endpointServer *nsmEndpointServer) error {
	listenEndpoint := endpointServer.endPointSocketPath
	logger := endpointServer.logger
	if err := tools.SocketCleanup(listenEndpoint); err != nil {
		return err
	}

	unix.Umask(socketMask)
	sock, err := net.Listen("unix", listenEndpoint)
	if err != nil {
		logger.Errorf("failure to listen on socket %s with error: %+v", listenEndpoint, err)
		return err
	}

	// Plugging Endpoint operations methods
	nseconnect.RegisterEndpointOperationsServer(endpointServer.grpcServer, endpointServer)
	logger.Infof("Starting Endpoint gRPC server listening on socket: %s", listenEndpoint)
	go func() {
		if err := endpointServer.grpcServer.Serve(sock); err != nil {
			logger.Fatalln("unable to start endpoint grpc server: ", listenEndpoint, err)
		}
	}()

	conn, err := tools.SocketOperationCheck(listenEndpoint)
	if err != nil {
		logger.Errorf("failure to communicate with the socket %s with error: %+v", listenEndpoint, err)
		return err
	}
	conn.Close()
	logger.Infof("Endpoint Server socket: %s is operational", listenEndpoint)

	// Wait for shutdown
	select {
	case <-endpointServer.stopChannel:
		logger.Infof("Server for socket %s received shutdown request", listenEndpoint)
	}
	endpointServer.stopChannel <- true

	return nil
}

// NewNSMEndpointServer registers and starts gRPC server which is listening for
// Network Service Endpoint advertise/remove calls and act accordingly
func NewNSMEndpointServer(p *Plugin) error {
	endpointServer := &nsmEndpointServer{
		logger:             p.Deps.Log,
		objectStore:        p.Deps.ObjectStore,
		k8sClient:          p.Deps.Client.GetClientset(),
		nsmClient:          p.Deps.Client.GetNSMClientset(),
		grpcServer:         grpc.NewServer(),
		endPointSocketPath: path.Join(EndpointSocketBaseDir, EndpointSocket),
		stopChannel:        make(chan bool),
		nsmNamespace:       p.namespace,
		nsmPodIPAddress:    p.nsmPodIPAddress,
	}

	var err error
	// Starting endpoint server, if it fails to start, inform Plugin by returning error
	go func() {
		err = startEndpointServer(endpointServer)
	}()

	return err
}
