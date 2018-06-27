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
	"time"

	"github.com/vishvananda/netns"

	"github.com/golang/glog"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/vishvananda/netlink"
	"google.golang.org/grpc"
)

const (
	// clientConnectionTimeout defines time the client waits for establishing connection with the server
	clientConnectionTimeout = time.Second * 60
)

var (
	clientSocket = flag.String("nsm-socket", "/var/lib/networkservicemesh/nsm.ligato.io.sock", "Location of NSM process client access socket")
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

	// Checking if NSM Client socket exists and of not crash init container
	_, err := os.Stat(*clientSocket)
	if err != nil {
		glog.Errorf("NSM Client: Failure to access NSM socket at %s with error: %+v, existing...", *clientSocket, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), clientConnectionTimeout)
	conn, err := dial(ctx, *clientSocket)
	if err != nil {
		glog.Errorf("NSM Client: Failure to communicate with the socket %s with error: %+v", *clientSocket, err)
		os.Exit(1)
	}
	nsmClient := nsmconnect.NewClientConnectionClient(conn)
	defer conn.Close()
	defer cancel()
	glog.Infof("NSM Client: Connection to NSM server on socket: %s succeeded.", *clientSocket)
	glog.Infof("NSM Client: Client API %+v", nsmClient)
	// Init related activities start here

	currentNamespace, err := netns.Get()
	if err != nil {
		glog.Errorf("NSM Client: Failure to get Pod's namespace with error: %+v", err)
		os.Exit(1)
	}
	glog.Infof("NSM Client: Pod's namespace is [%s]", currentNamespace.String())
	namespaceHandle, err := netlink.NewHandleAt(currentNamespace)
	if err != nil {
		glog.Errorf("NSM Client: Failure to get Pod's handle with error: %+v", err)
		os.Exit(1)
	}
	interfaces, err := namespaceHandle.LinkList()
	if err != nil {
		glog.Errorf("NSM Client: Failure to get Pod's interfaces with error: %+v", err)
	}
	glog.Info("NSM Client: Pod's interfaces:")
	for _, intf := range interfaces {
		glog.Infof("\t\tName: %s Type: %s", intf.Attrs().Name, intf.Type())
	}
	// Init related activities ends here
	glog.Info("NSM Client: Initialization is completed successfully, exiting...")
	os.Exit(0)
}
