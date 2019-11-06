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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

const (
	// NSETrackingIntervalDefault - default registry notification interval that NSE is still alive
	NSETrackingIntervalDefault = 2 * time.Minute
	// NSETrackingIntervalSecondsEnv - environment variable contains registry notification interval that NSE is still alive in seconds
	NSETrackingIntervalSecondsEnv = "NSE_TRACKING_INTERVAL"
)

type NSERegistryServer interface {
	registry.NetworkServiceRegistryServer
	RegisterNSEWithClient(ctx context.Context, request *registry.NSERegistration, client registry.NetworkServiceRegistryClient) (*registry.NSERegistration, error)
}
type registryServer struct {
	nsm       *nsmServer
	workspace *Workspace
}

func NewRegistryServer(nsm *nsmServer, workspace *Workspace) NSERegistryServer {
	return &registryServer{
		nsm:       nsm,
		workspace: workspace,
	}
}

func (es *registryServer) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	span := spanhelper.FromContext(ctx, "RegisterNSE")
	defer span.Finish()
	span.Logger().Infof("Received RegisterNSE request: %v", request)

	// Check if there is already Network Service Endpoint object with the same name, if there is
	// success will be returned to NSE, since it is a case of NSE pod coming back up.
	client, err := es.nsm.serviceRegistry.NseRegistryClient(span.Context())
	if err != nil {
		err = errors.Wrap(err, "attempt to connect to upstream registry failed with")
		span.LogError(err)
		return nil, err
	}

	reg, err := es.RegisterNSEWithClient(span.Context(), request, client)
	if err != nil {
		span.LogError(err)
		return reg, err
	}

	// Append to workspace...
	err = es.workspace.localRegistry.AppendNSERegRequest(es.workspace.name, reg)
	if err != nil {
		err = errors.Errorf("failed to store NSE into local registry service: %v", err)
		span.LogError(err)
		_, _ = client.RemoveNSE(span.Context(), &registry.RemoveNSERequest{NetworkServiceEndpointName: reg.GetNetworkServiceEndpoint().GetName()})
		return nil, err
	}
	span.LogObject("registration", reg)
	return reg, nil
}
func (es *registryServer) RegisterNSEWithClient(ctx context.Context, request *registry.NSERegistration, client registry.NetworkServiceRegistryClient) (*registry.NSERegistration, error) {
	// Some notes here:
	// 1)  Yes, we are overwriting anything we get for NetworkServiceManager
	//     from the NSE.  NSE's shouldn't specify NetworkServiceManager
	// 2)  We are not specifying Name, the nsmd-k8s will fill those
	//     in
	request.NetworkServiceManager = &registry.NetworkServiceManager{
		Url: es.nsm.serviceRegistry.GetPublicAPI(),
	}

	registration, err := client.RegisterNSE(ctx, request)
	if err != nil {
		err = errors.Wrap(err, "attempt to pass through from nsm to upstream registry failed with")
		logrus.Error(err)
		return nil, err
	}

	ep := es.nsm.model.GetEndpoint(registration.GetNetworkServiceEndpoint().GetName())
	modelEndpoint := &model.Endpoint{
		SocketLocation: es.workspace.NsmClientSocket(),
		Endpoint:       registration,
		Workspace:      es.workspace.Name(),
	}
	if ep == nil {
		es.nsm.model.AddEndpoint(ctx, modelEndpoint)
	}
	logrus.Infof("Received upstream NSERegitration: %v", registration)

	err = es.startNSETracking(registration)
	if err != nil {
		logrus.Infof("Error starting NSE tracking requests : %v", err)
	}

	return registration, nil
}

func (es *registryServer) BulkRegisterNSE(srv registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	<-srv.Context().Done()
	return nil
}

func (es *registryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	span := spanhelper.FromContext(ctx, "RemoveNSE")
	defer span.Finish()

	span.LogObject("request", request)

	// TODO make sure we track which registry server we got the RegisterNSE from so we can only allow a deletion
	// of what you advertised
	span.Logger().Infof("Received Endpoint Remove request: %+v", request)
	client, err := es.nsm.serviceRegistry.NseRegistryClient(span.Context())
	if err != nil {
		err = errors.Wrap(err, "attempt to pass through from nsm to upstream registry failed with")
		span.LogError(err)
		return nil, err
	}
	_, err = client.RemoveNSE(span.Context(), request)
	if err != nil {
		err = errors.Wrap(err, "attempt to pass through from nsm to upstream registry failed")
		span.LogError(err)
		return nil, err
	}
	es.nsm.model.DeleteEndpoint(span.Context(), request.GetNetworkServiceEndpointName())
	return &empty.Empty{}, nil
}

func (es *registryServer) Close() {

}

func (es *registryServer) startNSETracking(request *registry.NSERegistration) error {
	ctx, cancel := context.WithCancel(context.Background())

	client, err := es.nsm.serviceRegistry.NseRegistryClient(ctx)
	if err != nil {
		cancel()
		return errors.Errorf("cannot start NSE tracking : %v", err)
	}

	stream, err := client.BulkRegisterNSE(ctx)
	if err != nil {
		cancel()
		return errors.Errorf("cannot start NSE tracking : %v", err)
	}

	trackingInterval := NSETrackingIntervalDefault
	if interval := strings.TrimSpace(os.Getenv(NSETrackingIntervalSecondsEnv)); interval != "" {
		t, err := strconv.ParseInt(interval, 10, 32)
		if err != nil {
			logrus.Errorf("Cannot parse %s, use default value : %v", NSETrackingIntervalSecondsEnv, err)
		} else {
			trackingInterval = time.Duration(t) * time.Second
		}
	}

	go func() {
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(trackingInterval):
				err := stream.Send(request)
				if err != nil {
					logrus.Errorf("Error sending BulkRegisterNSE request : %v", err)
				}
			}
		}
	}()

	return nil
}
