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
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ligato/networkservicemesh/sdk/endpoint"
	"github.com/ligato/networkservicemesh/sdk/endpoint/composite"
	"github.com/sirupsen/logrus"
)

func main() {

	composite := composite.NewMonitorCompositeEndpoint(nil).SetNext(
		composite.NewIpamCompositeEndpoint(nil).SetNext(
			composite.NewConnectionCompositeEndpoint(nil)))

	nsmEndpoint, err := endpoint.NewNSMEndpoint(nil, nil, composite)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	nsmEndpoint.Start()
	defer nsmEndpoint.Delete()

	// Capture signals to cleanup before exiting
	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-c
		wg.Done()
	}()
	wg.Wait()
}
