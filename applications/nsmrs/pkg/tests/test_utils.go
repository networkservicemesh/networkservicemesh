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

// Package tests - unit tests for NSM Registry Server
package tests

import "github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"

func newTestNse(name, networkServiceName string) *registry.NSERegistration {
	return newTestNseWithPayload(name, networkServiceName, "IP")
}
func newTestNseWithPayload(name, networkServiceName, payload string) *registry.NSERegistration {
	return &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    networkServiceName,
			Payload: payload,
		},
		NetworkServiceManager: &registry.NetworkServiceManager{},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:    name,
			Payload: payload,
		},
	}
}
