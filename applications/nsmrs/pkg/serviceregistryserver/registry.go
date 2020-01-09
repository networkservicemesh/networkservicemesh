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

package serviceregistryserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

// NSERegistryService - service registering Network Service Endpoints
type NSERegistryService interface {
	RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error)
	BulkRegisterNSE(registry.NetworkServiceRegistry_BulkRegisterNSEServer) error
	RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error)
}

type nseRegistryService struct {
	cache NSERegistryCache
}

// NewNseRegistryService - creates NSE Registry service
func NewNseRegistryService(cache NSERegistryCache) NSERegistryService {
	return &nseRegistryService{
		cache: cache,
	}
}

// RegisterNSE - Registers NSE in cache
func (rs *nseRegistryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	span := spanhelper.FromContext(ctx, "Nsmrs.RegisterNSE")
	defer span.Finish()
	logger := span.Logger()

	logger.Infof("Received RegisterNSE(%v)", request)

	request = prepareNSERequest(request)

	_, err := rs.cache.AddNetworkServiceEndpoint(request)
	if err != nil {
		logger.Errorf("Error registering NSE: %v", err)
		return nil, err
	}

	logger.Infof("Returned from RegisterNSE: request: %v", request)
	return request, err
}

func (rs *nseRegistryService) BulkRegisterNSE(srv registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	span := spanhelper.FromContext(srv.Context(), "Nsmrs.BulkRegisterNSE")
	defer span.Finish()
	logger := span.Logger()

	for {
		request, err := srv.Recv()
		if err != nil {
			err = errors.Wrapf(err, "error receiving BulkRegisterNSE request : %v", err)
			return err
		}

		logger.Infof("Received BulkRegisterNSE request: %v", request)

		request = prepareNSERequest(request)

		_, err = rs.cache.UpdateNetworkServiceEndpoint(request)
		if err != nil {
			err = errors.Wrapf(err, "error processing BulkRegisterNSE request: %v", err)
			return err
		}
	}
}

// RemoveNSE - Removes NSE from cache and stops NSMgr monitor
func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	span := spanhelper.FromContext(ctx, "Nsmrs.RemoveNSE")
	defer span.Finish()
	logger := span.Logger()

	logger.Infof("Received RemoveNSE(%v)", request)

	nse, err := rs.cache.DeleteNetworkServiceEndpoint(request.NetworkServiceEndpointName)
	if err != nil {
		logger.Errorf("cannot remove Network Service Endpoint: %v", err)
		return &empty.Empty{}, err
	}

	logger.Infof("RemoveNSE done: %v", nse)
	return &empty.Empty{}, nil
}

func prepareNSERequest(request *registry.NSERegistration) *registry.NSERegistration {
	// Add public IP to NSM name to avoid name collision for different clusters
	nsmName := fmt.Sprintf("%s_%s", request.NetworkServiceManager.Name, request.NetworkServiceManager.Url)
	nsmName = strings.ReplaceAll(nsmName, ":", "_")
	request.NetworkServiceManager.Name = nsmName
	request.NetworkServiceEndpoint.NetworkServiceManagerName = nsmName

	return request
}
