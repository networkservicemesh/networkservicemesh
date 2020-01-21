// Copyright 2018, 2019 VMware, Inc.
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

package endpoint

import (
	"context"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/teris-io/shortid"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

// ConnectionEndpoint makes basic Mechanism selection for the incoming connection
type ConnectionEndpoint struct {
	mechanismType string
	// TODO - id doesn't seem to be used, and should be
	id *shortid.Shortid
}

// ParseVariable - parses var=value variable format.
func ParseVariable(variable string) (string, string, error) {
	pos := strings.Index(variable, "=")
	if pos == -1 {
		return "", "", errors.Errorf("variable passed are invalid")
	}
	return variable[:pos], variable[pos+1:], nil
}

// Request implements the request handler
// Consumes from ctx context.Context:
//	   Next
func (cce *ConnectionEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	err := request.IsValid()
	if err != nil {
		Log(ctx).Errorf("Request is not valid: %v", err)
		return nil, err
	}

	// FIXME: hardcoded
	mechanism, err := common.NewMechanism(cls.LOCAL, cce.mechanismType, cce.generateIfName(), "NSM Endpoint")
	if err != nil {
		Log(ctx).Errorf("Mechanism not created: %v", err)
		return nil, err
	}

	//mechanism.Type = "SRIOV_KERNEL_INTERFACE"

	inodeNum, err := tools.GetCurrentNS()
	if err != nil {
		return nil, err
	}

	environment := map[string]string{}
	for _, k := range os.Environ() {
		key, value, err := ParseVariable(k)
		if err != nil {
			return nil, err
		}
		environment[key] = value
	}

	params := map[string]string{
		"netnsInode": inodeNum,
	}

	if mechanism.Type == "SRIOV_KERNEL_INTERFACE" || mechanism.Type == "SRIOV_USERSPACE" {
		resourceEnvName, ok := environment["NSM_SRIOV_RESOURCE_NAME"]
		if !ok {
			return nil, errors.New("NSM_SRIOV_RESOURCE_NAME env variable missing")
		}
		pciAddress, ok := environment[resourceEnvName]
		if !ok {
			return nil, errors.Errorf("%s env variable missing", resourceEnvName)
		}

		params["PCIAddress"] = pciAddress
	}

	mechanism.Parameters = params

	request.GetConnection().Mechanism = mechanism

	if Next(ctx) != nil {
		return Next(ctx).Request(ctx, request)
	}
	return request.GetConnection(), nil
}

// Close implements the close handler
// Consumes from ctx context.Context:
//	   Next
func (cce *ConnectionEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (cce *ConnectionEndpoint) Name() string {
	return "connection"
}

func (cce *ConnectionEndpoint) generateIfName() string {
	ifName := "nsm" + cce.id.MustGenerate()
	ifName = strings.Replace(ifName, "-", "", -1)
	ifName = strings.Replace(ifName, "_", "", -1)

	return ifName
}

// NewConnectionEndpoint creates a ConnectionEndpoint
func NewConnectionEndpoint(configuration *common.NSConfiguration) *ConnectionEndpoint {
	// ensure the env variables are processed

	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	rand.Seed(time.Now().UTC().UnixNano())

	self := &ConnectionEndpoint{
		mechanismType: configuration.MechanismType,
		id:            shortid.MustNew(1, shortid.DefaultABC, rand.Uint64()),
	}
	if self.mechanismType == "" {
		self.mechanismType = kernel.MECHANISM
	}

	return self
}
