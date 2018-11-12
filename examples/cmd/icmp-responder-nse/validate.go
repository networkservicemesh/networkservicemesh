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

package main

// TODO - this should probably be moved to a library eventually

import (
	"fmt"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model/networkservice"
)

func ValidateNetworkServiceRequest(request *networkservice.NetworkServiceRequest) error {
	if request == nil {
		return fmt.Errorf("Request may not be nil")
	}
	if request.Connection == nil {
		return fmt.Errorf("Request.Connection may not be nil")
	}
	return ValidateConnection(request.Connection)
}

func ValidateConnection(connection *networkservice.Connection) error {
	if connection == nil {
		return fmt.Errorf("Connection may not be nil")
	}
	if connection.ConnectionId == "" {
		return fmt.Errorf("Connection.ConnectionId may not be empty")
	}
	// TODO we should validate the Network Service matches the NSE
	if connection.NetworkService == "" {
		return fmt.Errorf("Connection.NetworkService may not be empty: %v", connection)
	}
	return nil
}
