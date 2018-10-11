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

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmdataplane"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmregistration"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmvpp"
	"github.com/sirupsen/logrus"
)

const (
	dataplaneSocket = "/var/lib/networkservicemesh/nsm-vpp.dataplane.sock"
)

var (
	dataplane = flag.String("dataplane-socket", dataplaneSocket, "Location of the dataplane gRPC socket")
	debug     = flag.Bool("debug", false, "Enables extra tests which run during operation of nsm vpp dataplane controller")
)

var wg sync.WaitGroup

func startDataplaneAgent(vpp nsmvpp.Interface, stopCh chan struct{}) error {
	wg.Add(1)
	defer wg.Done()

	// Starting VPP Dataplane controller registration with NSM
	go nsmregistration.RegisterDataplane(vpp)

	errorCh := make(chan error)
	for {
		select {
		case <-stopCh:
			logrus.Infof("Received shutdown message, shutting down agent.")
			vpp.Shutdown()
			return nil
		case err := <-errorCh:
			return err
		}
	}
}

func main() {
	flag.Parse()
	// Initializing VPP
	vpp, err := nsmvpp.NEWVPPDataplane(*dataplane)
	if err != nil {
		logrus.Errorf("Failed to start VPP with error:%+v", err)
		os.Exit(1)
	}
	// Starting Dataplane gRPC server
	if err := nsmdataplane.StartDataplaneServer(vpp); err != nil {
		logrus.Errorf("Failed to start Dataplane gRPC server with error:%+v", err)
		os.Exit(1)
	}
	stopCh := make(chan struct{})
	go func() {
		if err := startDataplaneAgent(vpp, stopCh); err != nil {
			logrus.Errorf("Failed to start NSM VPP Dataplane controller with error:%+v", err)
			os.Exit(1)
		}
	}()
	if *debug {
		if err := vpp.Test(); err != nil {
			logrus.Errorf("Failed to test NSM VPP Dataplane controller with error:%+v", err)
			os.Exit(1)
		}
		// Introduce some fun
		go connectionBreaker(vpp)
	}
	// Capture signals to cleanup before exiting
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	for sig := range c {
		logrus.Infof("nsm-vpp-dataplane received termination signal %+v, cleaning up and terminating", sig)
		// Gracefully shutdown VPP and gRPC server
		stopCh <- struct{}{}
		wg.Wait()
		os.Exit(0)
	}
}

func waitForReady(f func() bool) error {
	ticker := time.NewTicker(1 * time.Second)
	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			if f() {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("Timeout for ready")
		}
	}
}

// This function is used to simulate VPP disconnect events and make sure
// reconnection logic and NSM registration/unregistration logic works
func connectionBreaker(v nsmvpp.Interface) {
	ticker := time.NewTicker(time.Minute * 5)
	for {
		select {
		case <-ticker.C:
			logrus.Info("\u2620 \u2620 \u2620  Killing connection with VPP, let's see if you can recover  \u2620 \u2620 \u2620")
			v.BreakConnection()
		}
	}
}
