// Copyright 2019 VMware, Inc.
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

package common

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	NsmBaseDirKey     = "NSM_BASEDIR"
	DefaultNsmBaseDir = "/var/lib/networkservicemesh/"
	// TODO Convert all the defaults to properly use NsmBaseDir
	DataplaneRegistrarSocketKey         = "DATAPLANE_REGISTRAR_SOCKET"
	DefaultDataplaneRegistrarSocket     = "/var/lib/networkservicemesh/nsm.dataplane-registrar.io.sock"
	DataplaneRegistrarSocketTypeKey     = "DATAPLANE_REGISTRAR_SOCKET_TYPE"
	DefaultDataplaneRegistrarSocketType = "unix"
	DataplaneSocketKey                  = "DATAPLANE_SOCKET"
	DefaultDataplaneSocket              = "/var/lib/networkservicemesh/nsm-vppagent.dataplane.sock"
	DataplaneSocketTypeKey              = "DATAPLANE_SOCKET_TYPE"
	DefaultDataplaneSocketType          = "unix"
	DataplaneNameKey                    = "DATAPLANE_NAME"
	DefaultDataplaneName                = "vppagent"
	DataplaneVPPAgentEndpointKey        = "VPPAGENT_ENDPOINT"
	DefaultVPPAgentEndpoint             = "localhost:9111"
	SrcIpEnvKey                         = "NSM_DATAPLANE_SRC_IP"
)

func DataplaneMain(createServer func(string, *EgressInterface) *grpc.Server) *dataplaneRegistration {
	start := time.Now()
	logrus.Info("Starting dataplane")

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

	dataplaneRegistrarSocketType, ok := os.LookupEnv(DataplaneRegistrarSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketTypeKey, DefaultDataplaneRegistrarSocketType)
		dataplaneRegistrarSocketType = DefaultDataplaneRegistrarSocketType
	}
	logrus.Infof("dataplaneRegistrarSocket: %s", dataplaneRegistrarSocketType)

	dataplaneSocket, ok := os.LookupEnv(DataplaneSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketKey, DefaultDataplaneSocket)
		dataplaneSocket = DefaultDataplaneSocket
	}
	logrus.Infof("dataplaneSocket: %s", dataplaneSocket)

	dataplaneSocketType, ok := os.LookupEnv(DataplaneSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketTypeKey, DefaultDataplaneSocketType)
		dataplaneSocketType = DefaultDataplaneSocketType
	}
	logrus.Infof("dataplaneSocketType: %s", dataplaneSocketType)

	dataplaneName, ok := os.LookupEnv(DataplaneNameKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneNameKey, DefaultDataplaneName)
		dataplaneName = DefaultDataplaneName
	}

	srcIpStr, ok := os.LookupEnv(SrcIpEnvKey)
	if !ok {
		logrus.Fatalf("Env variable %s must be set to valid srcIp for use for tunnels from this Pod.  Consider using downward API to do so.", SrcIpEnvKey)
		SetSrcIPFailed()
	}
	srcIp := net.ParseIP(srcIpStr)
	if srcIp == nil {
		logrus.Fatalf("Env variable %s must be set to a valid IP address, was set to %s", SrcIpEnvKey, srcIpStr)
		SetValidIPFailed()
	}

	egressInterface, err := NewEgressInterface(srcIp)
	if err != nil {
		logrus.Fatalf("Unable to find egress Interface: %s", err)
	}
	if err != nil {
		logrus.Fatalf("Unable to extract interface name for SrcIP: %s", srcIp)
		SetExtractIFNameFailed()
	}
	logrus.Infof("SrcIP: %s, IfaceName: %s, SrcIPNet: %s", srcIp, egressInterface.Name, egressInterface.SrcIPNet())

	logrus.Infof("dataplaneName: %s", dataplaneName)

	err = tools.SocketCleanup(dataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error cleaning up socket %s: %s", dataplaneSocket, err)
		SetSocketCleanFailed()
	}
	ln, err := net.Listen(dataplaneSocketType, dataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error listening on socket %s: %s ", dataplaneSocket, err)
		SetSocketListenFailed()
	}

	logrus.Info("Creating vppagent server")
	server := createServer(nsmBaseDir, egressInterface)
	go server.Serve(ln)
	logrus.Info("vppagent server serving")

	elapsed := time.Since(start)
	logrus.Debugf("Starting VPP Agent server took: %s", elapsed)

	logrus.Info("Dataplane Registrar Client")
	registrar := NewDataplaneRegistrarClient(dataplaneRegistrarSocketType, dataplaneRegistrarSocket)
	registration := registrar.Register(context.Background(), dataplaneName, dataplaneSocket, nil, nil)
	logrus.Info("Registered Dataplane Registrar Client")

	return registration
}
