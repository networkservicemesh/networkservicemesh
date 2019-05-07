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
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	dirty := false
	for _, arg := range os.Args[:1] {
		if arg == "--dirty" {
			dirty = true
		}
	}

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	composite := endpoint.NewCompositeEndpoint(
		endpoint.NewMonitorEndpoint(nil),
		endpoint.NewIpamEndpoint(nil),
		endpoint.NewConnectionEndpoint(nil))

	nsmEndpoint, err := endpoint.NewNSMEndpoint(nil, nil, composite)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	_ = nsmEndpoint.Start()
	if !dirty {
		defer func() { _ = nsmEndpoint.Delete() }()
	}

	// Capture signals to cleanup before exiting
	<-c
}
