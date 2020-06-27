// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package proxyregistryserver - forwards k8s commands from local registry to remote one
package proxyregistryserver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/clusterinfo"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
)

const (
	// NSRegistryForwarderLogPrefix - log prefix
	NSRegistryForwarderLogPrefix = "Network Service Registry Forwarder"
	// NSMRSAddressEnv - environment variable - address of Network Service Registry Server to forward NSE registry requests
	NSMRSAddressEnv = "NSMRS_ADDRESS"
	// NSMRSReconnectInterval - reconnect interval to NSMRS if connection refused
	NSMRSReconnectInterval = 15 * time.Second
)

type nseRegistryService struct {
	clusterInfoService clusterinfo.ClusterInfoServer
}

func newNseRegistryService(clusterInfoService clusterinfo.ClusterInfoServer) *nseRegistryService {
	return &nseRegistryService{
		clusterInfoService: clusterInfoService,
	}
}

func (rs *nseRegistryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	span := spanhelper.FromContext(ctx, "ProxyNsmgr.RegisterNSE")
	defer span.Finish()
	logger := span.Logger()

	logger.Infof("%s: received RegisterNSE(%v)", NSRegistryForwarderLogPrefix, request)

	nsmrsURL := os.Getenv(NSMRSAddressEnv)
	if strings.TrimSpace(nsmrsURL) == "" {
		err := errors.Errorf("NSMRS Address variable was not set")
		logger.Warnf("%s: Skipping Register NSE forwarding: %v", NSRegistryForwarderLogPrefix, err)
		return request, err
	}

	nodeConfiguration, cErr := rs.clusterInfoService.GetNodeIPConfiguration(span.Context(), &clusterinfo.NodeIPConfiguration{NodeName: request.NetworkServiceManager.Name})
	if cErr != nil {
		err := errors.Wrapf(cErr, "cannot get Network Service Manager's IP address: %s", cErr)
		logger.Errorf("%s: %v", NSRegistryForwarderLogPrefix, err)
		return nil, err
	}

	externalIP := nodeConfiguration.ExternalIP
	if externalIP == "" {
		externalIP = nodeConfiguration.InternalIP
	}
	// Swapping IP address to external (keep port)
	url := request.NetworkServiceManager.Url
	if idx := strings.Index(url, ":"); idx > -1 {
		externalIP += url[idx:]
	}
	request.NetworkServiceManager.Url = externalIP

	logger.Infof("%s: Prepared forwarding RegisterNSE request: %v", NSRegistryForwarderLogPrefix, request)

	remoteRegistry := nsmd.NewServiceRegistryAt(nsmrsURL)
	defer remoteRegistry.Stop()

	nseRegistryClient, err := remoteRegistry.NseRegistryClient(span.Context())
	if err != nil {
		logger.Warnf(fmt.Sprintf("%s: Cannot register network service endpoint in NSMRS: %v", NSRegistryForwarderLogPrefix, err))
		return request, err
	}

	_, err = nseRegistryClient.RegisterNSE(span.Context(), request)
	if err != nil {
		errIn := errors.Wrapf(err, "failed register NSE in NSMRS: %v", err)
		logger.Errorf("%s: %v", NSRegistryForwarderLogPrefix, errIn)
		return request, errIn
	}

	return request, nil
}

func (rs *nseRegistryService) BulkRegisterNSE(srv registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	span := spanhelper.FromContext(srv.Context(), "ProxyNsmgr.BulkRegisterNSE")
	defer span.Finish()
	logger := span.Logger()

	logger.Infof("%s: Forwarding Bulk Register NSE stream...", NSRegistryForwarderLogPrefix)

	nsmrsURL := os.Getenv(NSMRSAddressEnv)
	if strings.TrimSpace(nsmrsURL) == "" {
		err := errors.Errorf("NSMRS Address variable was not set")
		logger.Warnf("%s: Skipping Bulk Register NSE forwarding: %v", NSRegistryForwarderLogPrefix, err)
		return err
	}

	ctx, cancel := context.WithCancel(span.Context())
	defer cancel()

	remoteRegistry := nsmd.NewServiceRegistryAt(nsmrsURL)
	defer remoteRegistry.Stop()
	const maxReconnectAttempts = 10
	attempt := 0

	for {
		stream, err := requestBulkRegisterNSEStream(ctx, remoteRegistry, nsmrsURL)
		if err != nil {
			logger.Warnf("Cannot connect to Registry Server %s : %v", nsmrsURL, err)
			if attempt+1 == maxReconnectAttempts {
				return err
			}
			<-time.After(NSMRSReconnectInterval)
			attempt++
			continue
		}

		for {
			request, err := srv.Recv()
			if err != nil {
				err = errors.Wrapf(err, "error receiving BulkRegisterNSE request : %v", err)
				logger.Errorf("%s: %v", NSRegistryForwarderLogPrefix, err)
				return err
			}

			nodeConfiguration, cErr := rs.clusterInfoService.GetNodeIPConfiguration(span.Context(), &clusterinfo.NodeIPConfiguration{NodeName: request.NetworkServiceManager.Name})
			if cErr != nil {
				err := errors.Wrapf(cErr, "cannot get Network Service Manager's IP address: %s", cErr)
				logger.Errorf("%s: %v", NSRegistryForwarderLogPrefix, err)
				return err
			}

			externalIP := nodeConfiguration.ExternalIP
			if externalIP == "" {
				externalIP = nodeConfiguration.InternalIP
			}
			// Swapping IP address to external (keep port)
			url := request.NetworkServiceManager.Url
			if idx := strings.Index(url, ":"); idx > -1 {
				externalIP += url[idx:]
			}
			request.NetworkServiceManager.Url = externalIP

			logger.Infof("%s: Forward BulkRegisterNSE request: %v", NSRegistryForwarderLogPrefix, request)
			err = stream.Send(request)
			if err != nil {
				logger.Warnf("%s: error forwarding BulkRegisterNSE request to %s : %v", NSRegistryForwarderLogPrefix, nsmrsURL, err)
				break
			}
		}

		<-time.After(NSMRSReconnectInterval)
	}
}

func requestBulkRegisterNSEStream(ctx context.Context, remoteRegistry serviceregistry.ServiceRegistry, nsmrsURL string) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	nseRegistryClient, err := remoteRegistry.NseRegistryClient(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "error forwarding BulkRegisterNSE request to %s : %v", nsmrsURL, err)
	}

	stream, err := nseRegistryClient.BulkRegisterNSE(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "error forwarding BulkRegisterNSE request to %s : %v", nsmrsURL, err)
	}

	return stream, nil
}

func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	span := spanhelper.FromContext(ctx, "ProxyNsmgr.RemoveNSE")
	defer span.Finish()
	logger := span.Logger()

	logger.Infof("%s: Received RemoveNSE(%v)", NSRegistryForwarderLogPrefix, request)

	nsmrsURL := os.Getenv(NSMRSAddressEnv)
	if strings.TrimSpace(nsmrsURL) == "" {
		err := errors.Errorf("NSMRS Address variable was not set")
		logger.Warnf("%s: Skipping Register NSE forwarding: %v", NSRegistryForwarderLogPrefix, err)
		return &empty.Empty{}, err
	}

	remoteRegistry := nsmd.NewServiceRegistryAt(nsmrsURL)
	defer remoteRegistry.Stop()

	nseRegistryClient, err := remoteRegistry.NseRegistryClient(span.Context())
	if err != nil {
		logger.Warnf("%s: Cannot register network service endpoint in NSMRS: %v", NSRegistryForwarderLogPrefix, err)
		return &empty.Empty{}, err
	}

	if _, requestErr := nseRegistryClient.RemoveNSE(ctx, request); requestErr != nil {
		return &empty.Empty{}, requestErr
	}

	return &empty.Empty{}, nil
}
