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

package utils

import (
	"fmt"
	"time"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplane"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/testdataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"golang.org/x/net/context"
)

const (
	// dataplane default socket location
	dataplaneSocket = "/var/lib/networkservicemesh/dataplane.sock"
	// dataplaneConnectionTimeout defines a timeout to succeed connection to
	// the dataplane provider (seconds)
	dataplaneConnectionTimeout = 60 * time.Second
)

// ConnectPods builds dataplane connection between nsm client and nse providing
// required by the client network service
func ConnectPods(podName1, podName2, podNamespace1, podNamespace2 string) (*dataplane.Connection, error) {

	dataplaneConn, err := tools.SocketOperationCheck(dataplaneSocket)
	if err != nil {
		return nil, err
	}
	defer dataplaneConn.Close()

	dataplaneClient := testdataplane.NewBuildConnectClient(dataplaneConn)

	ctx, Cancel := context.WithTimeout(context.Background(), dataplaneConnectionTimeout)
	defer Cancel()
	buildConnectRequest := &testdataplane.BuildConnectRequest{
		SourcePod: &testdataplane.Pod{
			Metadata: &testdataplane.Metadata{
				Name:      podName1,
				Namespace: podNamespace1,
			},
		},
		DestinationPod: &testdataplane.Pod{
			Metadata: &testdataplane.Metadata{
				Name:      podName2,
				Namespace: podNamespace2,
			},
		},
	}
	buildConnectRepl, err := dataplaneClient.RequestBuildConnect(ctx, buildConnectRequest)
	if err != nil {
		if buildConnectRepl != nil {
			return nil, fmt.Errorf("%+v with additional information: %s", err, buildConnectRepl.BuildError)
		}
		return nil, err
	}

	return buildConnectRepl.Connection, nil
}

// CleanupPodDataplane cleans up from the given pod previsouly injected  dataplane
// interfaces
func CleanupPodDataplane(podName string, podNamespace string, podType testdataplane.NSMPodType) error {

	dataplaneConn, err := tools.SocketOperationCheck(dataplaneSocket)
	if err != nil {
		return err
	}
	defer dataplaneConn.Close()

	dataplaneClient := testdataplane.NewDeleteConnectClient(dataplaneConn)

	ctx, Cancel := context.WithTimeout(context.Background(), dataplaneConnectionTimeout)
	defer Cancel()
	deleteConnectRequest := &testdataplane.DeleteConnectRequest{
		Pod: &testdataplane.Pod{
			Metadata: &testdataplane.Metadata{
				Name:      podName,
				Namespace: podNamespace,
			},
		},
		PodType: podType,
	}
	deleteConnectRepl, err := dataplaneClient.RequestDeleteConnect(ctx, deleteConnectRequest)
	if err != nil {
		if deleteConnectRepl != nil {
			return fmt.Errorf("%+v with additional information: %s", err, deleteConnectRepl.DeleteError)
		}
		return err
	}

	return nil
}
