// Copyright (c) 2018-2019 Cisco and/or its affiliates.
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

package nsminit

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/sriovkernel"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"

	"github.com/networkservicemesh/networkservicemesh/sdk/client"
)

type nsmClientApp struct {
	configuration *common.NSConfiguration
}

func (c *nsmClientApp) Run() {
	closer := jaeger.InitJaeger("nsm-init")
	defer func() { _ = closer.Close() }()

	span := spanhelper.FromContext(context.Background(), "RequestNetworkService")
	defer span.Finish()

	c.configuration = c.configuration.FromEnv()
	if c.configuration.PodName == "" {
		podName, err := tools.GetCurrentPodNameFromHostname()
		if err != nil {
			logrus.Infof("failed to get current pod name from hostname: %v", err)
		} else {
			c.configuration.PodName = podName
		}
	}
	if c.configuration.Namespace == "" {
		c.configuration.Namespace = common.GetNamespace()
	}

	clientList, err := client.NewNSMClientList(span.Context(), c.configuration)
	if err != nil {
		span.Finish()
		_ = closer.Close()
		logrus.Fatalf("nsm client: Unable to create the NSM client %v", err)
		return
	}
	// TODO: fix hardcoded mechanism request change here
	err = clientList.ConnectRetry(span.Context(), "nsm", sriovkernel.MECHANISM, "Primary interface", client.ConnectionRetry, client.RequestDelay)
	if err != nil {
		span.Finish()
		_ = closer.Close()
		logrus.Fatalf("nsm client: Unable to establish connection with network service")
		return
	}
	logrus.Info("nsm client: initialization is completed successfully")
}

// NewNSMClientApp - creates a client application.
func NewNSMClientApp(configration *common.NSConfiguration) NSMApp {
	return &nsmClientApp{
		configuration: configration,
	}
}
