// Copyright (c) 2019 Cisco Systems, Inc.
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

package main

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/prefixcollector"
	k8s_utils "github.com/networkservicemesh/networkservicemesh/k8s/pkg/utils"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

func main() {
	logrus.Info("Starting prefix service...")
	utils.PrintAllEnv(logrus.StandardLogger())
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	closer := jaeger.InitJaeger("prefix-service")
	defer func() { _ = closer.Close() }()

	span := spanhelper.FromContext(context.Background(), "Start prefix service")
	defer span.Finish()

	span.Logger().Printf("Building Kubernetes clientset...")
	_, config, err := k8s_utils.NewClientSet()
	if err != nil {
		span.LogError(err)
		span.Logger().Fatalln("Failed to build Kubernetes clientset: ", err)
	}

	span.Logger().Infof("Starting prefix service...")
	err = prefixcollector.NewExcludePrefixServer(config)
	if err != nil {
		span.Logger().Fatalln(err)
	}

	span.Finish() // exclude main cycle run time from span timing
	<-c
}
