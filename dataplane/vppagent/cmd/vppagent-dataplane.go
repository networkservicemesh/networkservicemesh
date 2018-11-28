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
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ligato/networkservicemesh/dataplane/impl/dataplaneregistrarclient"
	"github.com/ligato/networkservicemesh/dataplane/vppagent/pkg/vppagent"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
)

const (
	NsmBaseDirKey     = "NSM_BASEDIR"
	DefaultNsmBaseDir = "/var/lib/networkservicemesh/"
	// TODO Convert all the defaults to properly use NsmBaseDir
	DataplaneRegistrarSocketKey     = "DATAPLANE_REGISTRAR_SOCKET"
	DefaultDataplaneRegistrarSocket = "/var/lib/networkservicemesh/nsm.dataplane-registrar.io.sock"
	DataplaneSocketKey              = "DATAPLANE_SOCKET"
	DefaultDataplaneSocket          = "/var/lib/networkservicemesh/nsm-vppagent.dataplane.sock"
	DataplaneNameKey                = "DATAPLANE_NAME"
	DefaultDataplaneName            = "vppagent"
	DataplaneVPPAgentEndpointKey    = "VPPAGENT_ENDPOINT"
	DefaultVPPAgentEndpoint         = "localhost:9111"
	SrcIpEnvKey                     = "NSM_DATAPLANE_SRC_IP"
)

func main() {

	// Capture signals to cleanup before exiting
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	logrus.Info("Starting vppagent-dataplane")

	nsmBaseDir, ok := os.LookupEnv(NsmBaseDirKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", NsmBaseDirKey, DefaultNsmBaseDir)
		nsmBaseDir = DefaultNsmBaseDir
	}
	logrus.Infof("nsmBaseDir: %s", nsmBaseDir)

	dataplaneRegistrarSocket, ok := os.LookupEnv(DataplaneRegistrarSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketKey, DefaultDataplaneRegistrarSocket)
		dataplaneRegistrarSocket = DefaultDataplaneRegistrarSocket
	}
	logrus.Infof("dataplaneRegistrarSocket: %s", dataplaneRegistrarSocket)

	dataplaneSocket, ok := os.LookupEnv(DataplaneSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketKey, DefaultDataplaneSocket)
		dataplaneSocket = DefaultDataplaneSocket
	}
	logrus.Infof("dataplaneSocket: %s", dataplaneSocket)

	vppAgentEndpoint, ok := os.LookupEnv(DataplaneVPPAgentEndpointKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneVPPAgentEndpointKey, DefaultVPPAgentEndpoint)
		vppAgentEndpoint = DefaultVPPAgentEndpoint
	}
	logrus.Infof("vppAgentEndpoint: %s", vppAgentEndpoint)

	dataplaneName, ok := os.LookupEnv(DataplaneNameKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneNameKey, DefaultDataplaneName)
		dataplaneName = DefaultDataplaneName
	}

	srcIpStr, ok := os.LookupEnv(SrcIpEnvKey)
	if !ok {
		logrus.Fatalf("Env variable %s must be set to valid srcIp for use for tunnels from this Pod.  Consider using downward API to do so.", SrcIpEnvKey)
	}
	srcIp := net.ParseIP(srcIpStr)
	if srcIp == nil {
		logrus.Fatalf("Env variable %s must be set to a valid IP address, was set to %s", SrcIpEnvKey, srcIpStr)
	}
	ifaceName, srcIpNet, err := MgmtIface(srcIp)
	if err != nil {
		logrus.Fatalf("Unable to extract interface name for SrcIP: %s", srcIp)
	}
	logrus.Infof("SrcIP: %s, IfaceName: %s, SrcIPNet: %s", srcIp, ifaceName, srcIpNet)

	logrus.Infof("dataplaneName: %s", dataplaneName)

	err = tools.SocketCleanup(dataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error cleaning up socket %s: %s", dataplaneSocket, err)
	}
	ln, err := net.Listen("unix", dataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error listening on socket %s: %s ", dataplaneSocket, err)
	}
	logrus.Info("Creating vppagent server")
	server := vppagent.NewServer(vppAgentEndpoint, nsmBaseDir, srcIp, *srcIpNet, ifaceName)
	go server.Serve(ln)
	logrus.Info("vppagent server serving")

	logrus.Info("Dataplane Registrar Client")
	registrar := dataplaneregistrarclient.NewDataplaneRegistrarClient(dataplaneRegistrarSocket)
	registration := registrar.Register(context.Background(), dataplaneName, dataplaneSocket, nil, nil)
	logrus.Info("Registered Dataplane Registrar Client")

	select {
	case <-c:
		logrus.Info("Closing Dataplane Registration")
		registration.Close()
	}
}

func MgmtIface(srcIp net.IP) (string, *net.IPNet, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", nil, err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", nil, err
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.Equal(srcIp) {
					return iface.Name, v, nil
				}
			default:
				return "", nil, fmt.Errorf("Type of addr not net.IPNET")
			}
		}
	}
	return "", nil, nil
}
