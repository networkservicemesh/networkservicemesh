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
	NSMBaseDirKey     = "NSM_BASEDIR"
	DefaultNSMBaseDir = "/var/lib/networkservicemesh/"
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
	SrcIPEnvKey                         = "NSM_DATAPLANE_SRC_IP"
)

type dataplaneConfig struct {
	name                string
	nsmBaseDir          string
	registrarSocket     string
	registrarSocketType string
	dataplaneSocket     string
	dataplaneSocketType string
	srcIP               []byte
	egressInterface     *EgressInterface
}

func getDataplaneConfig() *dataplaneConfig {
	dpCfg := new(dataplaneConfig)
	var ok bool
	var err error

	dpCfg.name, ok = os.LookupEnv(DataplaneNameKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneNameKey, DefaultDataplaneName)
		dpCfg.name = DefaultDataplaneName
	}

	logrus.Infof("Starting dataplane - %s", dpCfg.name)

	dpCfg.nsmBaseDir, ok = os.LookupEnv(NSMBaseDirKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", NSMBaseDirKey, DefaultNSMBaseDir)
		dpCfg.nsmBaseDir = DefaultNSMBaseDir
	}
	logrus.Infof("nsmBaseDir: %s", dpCfg.nsmBaseDir)

	dpCfg.registrarSocket, ok = os.LookupEnv(DataplaneRegistrarSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketKey, DefaultDataplaneRegistrarSocket)
		dpCfg.registrarSocket = DefaultDataplaneRegistrarSocket
	}
	logrus.Infof("registrarSocket: %s", dpCfg.registrarSocket)

	dpCfg.registrarSocketType, ok = os.LookupEnv(DataplaneRegistrarSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketTypeKey, DefaultDataplaneRegistrarSocketType)
		dpCfg.registrarSocketType = DefaultDataplaneRegistrarSocketType
	}
	logrus.Infof("registrarSocket: %s", dpCfg.registrarSocketType)

	dpCfg.dataplaneSocket, ok = os.LookupEnv(DataplaneSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketKey, DefaultDataplaneSocket)
		dpCfg.dataplaneSocket = DefaultDataplaneSocket
	}
	logrus.Infof("dataplaneSocket: %s", dpCfg.dataplaneSocket)

	dpCfg.dataplaneSocketType, ok = os.LookupEnv(DataplaneSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketTypeKey, DefaultDataplaneSocketType)
		dpCfg.dataplaneSocketType = DefaultDataplaneSocketType
	}
	logrus.Infof("dataplaneSocketType: %s", dpCfg.dataplaneSocketType)

	srcIPStr, ok := os.LookupEnv(SrcIPEnvKey)
	if !ok {
		logrus.Fatalf("Env variable %s must be set to valid srcIP for use for tunnels from this Pod.  Consider using downward API to do so.", SrcIPEnvKey)
		SetSrcIPFailed()
	}
	dpCfg.srcIP = net.ParseIP(srcIPStr)
	if dpCfg.srcIP == nil {
		logrus.Fatalf("Env variable %s must be set to a valid IP address, was set to %s", SrcIPEnvKey, srcIPStr)
		SetValidIPFailed()
	}

	dpCfg.egressInterface, err = NewEgressInterface(dpCfg.srcIP)
	if err != nil {
		logrus.Fatalf("Unable to find egress Interface: %s", err)
	}
	if err != nil {
		logrus.Fatalf("Unable to extract interface name for SrcIP: %s", dpCfg.srcIP)
		SetExtractIFNameFailed()
	}
	logrus.Infof("SrcIP: %s, IfaceName: %s, SrcIPNet: %s", dpCfg.srcIP, dpCfg.egressInterface.Name, dpCfg.egressInterface.SrcIPNet())

	return dpCfg
}

func DataplaneMain(createServer func(string, *EgressInterface) *grpc.Server) *dataplaneRegistration {
	start := time.Now()

	dataplaneCfg := getDataplaneConfig()

	err := tools.SocketCleanup(dataplaneCfg.dataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error cleaning up socket %s: %s", dataplaneCfg.dataplaneSocket, err)
		SetSocketCleanFailed()
	}

	ln, err := net.Listen(dataplaneCfg.dataplaneSocketType, dataplaneCfg.dataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error listening on socket %s: %s ", dataplaneCfg.dataplaneSocket, err)
		SetSocketListenFailed()
	}

	logrus.Infof("Creating %s server...", dataplaneCfg.name)
	server := createServer(dataplaneCfg.nsmBaseDir, dataplaneCfg.egressInterface)
	go server.Serve(ln)
	logrus.Infof("%s server serving", dataplaneCfg.name)

	elapsed := time.Since(start)
	logrus.Debugf("Starting the %s dataplane server took: %s", dataplaneCfg.name, elapsed)

	logrus.Info("Creating Dataplane Registrar Client...")
	registrar := NewDataplaneRegistrarClient(dataplaneCfg.registrarSocketType, dataplaneCfg.registrarSocket)
	registration := registrar.Register(context.Background(), dataplaneCfg.name, dataplaneCfg.dataplaneSocket, nil, nil)
	logrus.Info("Registered Dataplane Registrar Client")

	return registration
}
