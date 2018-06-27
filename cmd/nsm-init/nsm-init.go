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
	"context"
	"flag"
	"net"
	"os"
	"path"
	"time"

	"github.com/vishvananda/netns"

	"github.com/ligato/networkservicemesh/nsmdp"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"google.golang.org/grpc"
)

const (
	// clientConnectionTimeout defines time the client waits for establishing connection with the server
	clientConnectionTimeout = time.Second * 60
)

var (
	clientSocketPath     = path.Join(nsmdp.SocketBaseDir, nsmdp.ServerSock)
	clientSocketUserPath = flag.String("nsm-socket", "", "Location of NSM process client access socket")
)

func dial(ctx context.Context, unixSocketPath string) (*grpc.ClientConn, error) {
	c, err := grpc.DialContext(ctx, unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	return c, err
}

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")

	// Checking if nsm client socket exists and of not crash init container
	clientSocket := clientSocketPath
	if clientSocketUserPath != nil {
		clientSocket = *clientSocketUserPath
	}
	_, err := os.Stat(clientSocket)
	if err != nil {
		logrus.Errorf("nsm client: failure to access nsm socket at %s with error: %+v, existing...", clientSocket, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), clientConnectionTimeout)
	conn, err := dial(ctx, clientSocket)
	if err != nil {
		logrus.Errorf("nsm client: failure to communicate with the socket %s with error: %+v", clientSocket, err)
		os.Exit(1)
	}
	nsmClient := nsmconnect.NewClientConnectionClient(conn)
	defer conn.Close()
	defer cancel()
	logrus.Infof("nsm client: connection to nsm server on socket: %s succeeded.", clientSocket)
	logrus.Infof("nsm client: client api %+v", nsmClient)
	// Init related activities start here

	currentNamespace, err := netns.Get()
	if err != nil {
		logrus.Errorf("nsm client: failure to get pod's namespace with error: %+v", err)
		os.Exit(1)
	}
	logrus.Infof("nsm client: pod's namespace is [%s]", currentNamespace.String())
	namespaceHandle, err := netlink.NewHandleAt(currentNamespace)
	if err != nil {
		logrus.Errorf("nsm client: failure to get pod's handle with error: %+v", err)
		os.Exit(1)
	}
	interfaces, err := namespaceHandle.LinkList()
	if err != nil {
		logrus.Errorf("nsm client: pailure to get pod's interfaces with error: %+v", err)
	}
	logrus.Info("nsm client: pod's interfaces:")
	for _, intf := range interfaces {
		logrus.Infof("Name: %s Type: %s", intf.Attrs().Name, intf.Type())
	}
	// Init related activities ends here
	logrus.Info("nsm client: initialization is completed successfully, exiting...")
	os.Exit(0)
}
