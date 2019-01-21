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

	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint/composite"
	"github.com/sirupsen/logrus"
)

func main() {

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.TraceLevel)

	configuration := &common.NSConfiguration{
		MechanismType: "mem",
	}

	composite := composite.NewMonitorCompositeEndpoint(nil).SetNext(
		newVppAgentAclComposite(configuration).SetNext(
			newVppAgentXConnComposite(configuration).SetNext(
				composite.NewClientCompositeEndpoint(configuration).SetNext(
					composite.NewConnectionCompositeEndpoint(configuration)))))

	nsmEndpoint, err := endpoint.NewNSMEndpoint(nil, configuration, composite)
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
