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
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

type NSMDataplane interface {
	dataplane.DataplaneServer
	dataplane.MechanismsMonitorServer
	Init(*DataplaneConfig) error
}

// TODO Convert all the defaults to properly use NsmBaseDir
const (
	NSMBaseDirKey                        = "NSM_BASEDIR"
	NSMBaseDirDefault                    = "/var/lib/networkservicemesh/"
	DataplaneRegistrarSocketKey          = "DATAPLANE_REGISTRAR_SOCKET"
	DataplaneRegistrarSocketDefault      = "/var/lib/networkservicemesh/nsm.dataplane-registrar.io.sock"
	DataplaneRegistrarSocketTypeKey      = "DATAPLANE_REGISTRAR_SOCKET_TYPE"
	DataplaneRegistrarSocketTypeDefault  = "unix"
	DataplaneMetricsEnabledKey           = "METRICS_COLLECTOR_ENABLED"
	DataplaneMetricsEnabledDefault       = true
	DataplaneMetricsRequestPeriodKey     = "METRICS_COLLECTOR_REQUEST_PERIOD"
	DataplaneMetricsRequestPeriodDefault = time.Second * 2
	DataplaneNameKey                     = "DATAPLANE_NAME"
	DataplaneNameDefault                 = "vppagent"
	DataplaneSocketKey                   = "DATAPLANE_SOCKET"
	DataplaneSocketDefault               = "/var/lib/networkservicemesh/nsm-vppagent.dataplane.sock"
	DataplaneSocketTypeKey               = "DATAPLANE_SOCKET_TYPE"
	DataplaneSocketTypeDefault           = "unix"
	DataplaneSrcIPKey                    = "NSM_DATAPLANE_SRC_IP"
)

// DataplaneConfig keeps the common configuration for a forwarding plane
type DataplaneConfig struct {
	Name                    string
	NSMBaseDir              string
	RegistrarSocket         string
	RegistrarSocketType     string
	DataplaneSocket         string
	DataplaneSocketType     string
	MechanismsUpdateChannel chan *Mechanisms
	Mechanisms              *Mechanisms
	MetricsEnabled          bool
	MetricsPeriod           time.Duration
	SrcIP                   net.IP
	EgressInterface         EgressInterfaceType
	GRPCserver              *grpc.Server
	Monitor                 monitor_crossconnect.MonitorServer
	Listener                net.Listener
}

// Mechanisms is a message used to communicate any changes in operational parameters and constraints
type Mechanisms struct {
	RemoteMechanisms []*remote.Mechanism
	LocalMechanisms  []*local.Mechanism
}

func createDataplaneConfig(dataplaneGoals *DataplaneProbeGoals) *DataplaneConfig {
	cfg := &DataplaneConfig{}
	var ok bool

	cfg.Name, ok = os.LookupEnv(DataplaneNameKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneNameKey, DataplaneNameDefault)
		cfg.Name = DataplaneNameDefault
	}
	logrus.Infof("Starting dataplane - %s", cfg.Name)

	cfg.DataplaneSocket, ok = os.LookupEnv(DataplaneSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketKey, DataplaneSocketDefault)
		cfg.DataplaneSocket = DataplaneSocketDefault
	}
	logrus.Infof("DataplaneSocket: %s", cfg.DataplaneSocket)

	err := tools.SocketCleanup(cfg.DataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error cleaning up socket %s: %s", cfg.DataplaneSocket, err)
	} else {
		dataplaneGoals.SetSocketCleanReady()
	}

	cfg.DataplaneSocketType, ok = os.LookupEnv(DataplaneSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketTypeKey, DataplaneSocketTypeDefault)
		cfg.DataplaneSocketType = DataplaneSocketTypeDefault
	}
	logrus.Infof("DataplaneSocketType: %s", cfg.DataplaneSocketType)

	cfg.NSMBaseDir, ok = os.LookupEnv(NSMBaseDirKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", NSMBaseDirKey, NSMBaseDirDefault)
		cfg.NSMBaseDir = NSMBaseDirDefault
	}
	logrus.Infof("NSMBaseDir: %s", cfg.NSMBaseDir)

	cfg.RegistrarSocket, ok = os.LookupEnv(DataplaneRegistrarSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketKey, DataplaneRegistrarSocketDefault)
		cfg.RegistrarSocket = DataplaneRegistrarSocketDefault
	}
	logrus.Infof("RegistrarSocket: %s", cfg.RegistrarSocket)

	cfg.RegistrarSocketType, ok = os.LookupEnv(DataplaneRegistrarSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketTypeKey, DataplaneRegistrarSocketTypeDefault)
		cfg.RegistrarSocketType = DataplaneRegistrarSocketTypeDefault
	}
	logrus.Infof("RegistrarSocketType: %s", cfg.RegistrarSocketType)

	cfg.GRPCserver = tools.NewServer()

	cfg.Monitor = monitor_crossconnect.NewMonitorServer()
	crossconnect.RegisterMonitorCrossConnectServer(cfg.GRPCserver, cfg.Monitor)

	cfg.MetricsEnabled = DataplaneMetricsEnabledDefault
	val, ok := os.LookupEnv(DataplaneMetricsEnabledKey)
	if ok {
		res, err := strconv.ParseBool(val)
		if err == nil {
			cfg.MetricsEnabled = res
		}
	}
	logrus.Infof("MetricsEnabled: %v", cfg.MetricsEnabled)

	if cfg.MetricsEnabled {
		cfg.MetricsPeriod = DataplaneMetricsRequestPeriodDefault
		if val, ok = os.LookupEnv(DataplaneMetricsRequestPeriodKey); ok {
			parsedPeriod, err := time.ParseDuration(val)
			if err == nil {
				cfg.MetricsPeriod = parsedPeriod
			}
		}
		logrus.Infof("MetricsPeriod: %v ", cfg.MetricsPeriod)
	}

	srcIPStr, ok := os.LookupEnv(DataplaneSrcIPKey)
	if !ok {
		logrus.Fatalf("Env variable %s must be set to valid srcIP for use for tunnels from this Pod.  Consider using downward API to do so.", DataplaneSrcIPKey)
	} else {
		dataplaneGoals.SetSrcIPReady()
	}
	cfg.SrcIP = net.ParseIP(srcIPStr)
	if cfg.SrcIP == nil {
		logrus.Fatalf("Env variable %s must be set to a valid IP address, was set to %s", DataplaneSrcIPKey, srcIPStr)
	} else {
		dataplaneGoals.SetValidIPReady()
	}
	cfg.EgressInterface, err = NewEgressInterface(cfg.SrcIP)
	if err != nil {
		logrus.Fatalf("Unable to find egress Interface: %s", err)
	} else {
		dataplaneGoals.SetNewEgressIFReady()
	}
	logrus.Infof("SrcIP: %s, IfaceName: %s, SrcIPNet: %s", cfg.SrcIP, cfg.EgressInterface.Name(), cfg.EgressInterface.SrcIPNet())

	return cfg
}

// CreateDataplane creates new Dataplane Registrar client
func CreateDataplane(dp NSMDataplane, dataplaneGoals *DataplaneProbeGoals) *DataplaneRegistration {
	start := time.Now()
	// Populate common configuration
	config := createDataplaneConfig(dataplaneGoals)

	// Initialize the dataplane
	err := dp.Init(config)
	if err != nil {
		logrus.Fatalf("Dataplane initialization failed: %s ", err)
	}

	// Verify the configuration is populated
	if !sanityCheckConfig(config) {
		logrus.Fatalf("Dataplane configuration sanity check failed: %s ", err)
	}

	// Prepare the gRPC server
	config.Listener, err = net.Listen(config.DataplaneSocketType, config.DataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error listening on socket %s: %s ", config.DataplaneSocket, err)
	} else {
		dataplaneGoals.SetSocketListenReady()
	}
	dataplane.RegisterDataplaneServer(config.GRPCserver, dp)
	dataplane.RegisterMechanismsMonitorServer(config.GRPCserver, dp)

	// Start the server
	logrus.Infof("Creating %s server...", config.Name)
	go func() {
		_ = config.GRPCserver.Serve(config.Listener)
	}()
	logrus.Infof("%s server serving", config.Name)

	logrus.Debugf("Starting the %s dataplane server took: %s", config.Name, time.Since(start))

	logrus.Info("Creating Dataplane Registrar Client...")
	registrar := NewDataplaneRegistrarClient(config.RegistrarSocketType, config.RegistrarSocket)
	registration := registrar.Register(context.Background(), config.Name, config.DataplaneSocket, nil, nil)
	logrus.Info("Registered Dataplane Registrar Client")

	return registration
}

func sanityCheckConfig(dataplaneConfig *DataplaneConfig) bool {
	return len(dataplaneConfig.Name) > 0 &&
		len(dataplaneConfig.NSMBaseDir) > 0 &&
		len(dataplaneConfig.RegistrarSocket) > 0 &&
		len(dataplaneConfig.RegistrarSocketType) > 0 &&
		len(dataplaneConfig.DataplaneSocket) > 0 &&
		len(dataplaneConfig.DataplaneSocketType) > 0
}

// SanityCheckConnectionType checks whether the forwarding plane supports the connection type in the request
func SanityCheckConnectionType(mechanisms *Mechanisms, crossConnect *crossconnect.CrossConnect) error {
	localFound, remoteFound := false, false
	/* Verify local mechanisms */
	for _, mech := range mechanisms.LocalMechanisms {
		if crossConnect.GetLocalSource().GetMechanism().GetType() == mech.GetType() || crossConnect.GetLocalDestination().GetMechanism().GetType() == mech.GetType() {
			localFound = true
			break
		}
	}
	/* Verify remote mechanisms */
	for _, mech := range mechanisms.RemoteMechanisms {
		if crossConnect.GetRemoteSource().GetMechanism().GetType() == mech.GetType() || crossConnect.GetRemoteDestination().GetMechanism().GetType() == mech.GetType() {
			remoteFound = true
			break
		}
	}
	/* If none of them matched, mechanism is not supported by the forwarding plane */
	if !localFound && !remoteFound {
		return fmt.Errorf("connection mechanism type not supported by the forwarding plane")
	}
	return nil
}
