// Copyright 2019 Red Hat, Inc.
// Copyright (c) 2019 Cisco and/or its affiliates.
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
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/spanhelper"
	"os"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var version string

func main() {
	logrus.Info("Starting nsmdp-k8s...")
	logrus.Infof("Version %v", version)
	// Capture signals to cleanup before exiting
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	if tools.IsOpentracingEnabled() {
		tracer, closer := tools.InitJaeger("nsmdp")
		defer func() {
			if err := closer.Close(); err != nil {
				logrus.Errorf("An error during closing: %v", err)
			}
		}()
		opentracing.SetGlobalTracer(tracer)
	}

	span := spanhelper.SpanHelperFromContext(context.Background(), "NSMgr.Device.Plugin")
	defer span.Finish() // Finish since it will be root for all events.

	serviceRegistry := nsmd.NewServiceRegistry()
	span.LogObject("registry.at", serviceRegistry.GetPublicAPI())

	err := NewNSMDeviceServer(span.Context(), serviceRegistry)

	if err != nil {
		span.LogError(err)
		logrus.Errorf("failed to start server: %v", err)
		os.Exit(1)
	}

	span.Logger().Info("nsmdp: successfully started")
	span.Finish()
	<-c
}
