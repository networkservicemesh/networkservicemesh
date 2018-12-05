// Copyright 2018 VMware, Inc.
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

package main

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/sirupsen/logrus"
)

func (ns *vppagentNetworkService) CompleteConnection(request *networkservice.NetworkServiceRequest, outgoingConnection *connection.Connection) (*connection.Connection, error) {
	err := request.IsValid()
	if err != nil {
		return nil, err
	}
	mechanism, err := connection.NewMechanism(connection.MechanismType_MEM_INTERFACE, "nsm"+request.GetConnection().GetId(), "")
	if err != nil {
		return nil, err
	}

	connection := &connection.Connection{
		Id:             request.GetConnection().GetId(),
		NetworkService: request.GetConnection().GetNetworkService(),
		Mechanism:      mechanism,
		Context:        outgoingConnection.GetContext(),
	}
	err = connection.IsComplete()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return connection, nil
}
