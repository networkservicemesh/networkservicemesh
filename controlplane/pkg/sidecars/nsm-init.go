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

package sidecars

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
)

type nsmClientApp struct {
}

func (c *nsmClientApp) Run() {
	tracer, closer := tools.InitJaeger("nsm-init")
	opentracing.SetGlobalTracer(tracer)
	defer func() { _ = closer.Close() }()

	clientList, err := client.NewNSMClientList(context.Background(), nil)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client %v", err)
	}

	if err := clientList.Connect(context.TODO(), "nsm", "kernel", "Primary interface"); err != nil {
		logrus.Fatalf("Client connect failed with error: %v", err)
	}
	logrus.Info("nsm client: initialization is completed successfully")
}

// NewNSMClientApp - creates a client application.
func NewNSMClientApp() NSMApp {
	return &nsmClientApp{}
}
