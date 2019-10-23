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
	"strconv"
	"time"

	"github.com/networkservicemesh/networkservicemesh/sdk/compat"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/sdk/monitor/crossconnect"
)

type NSMForwarder interface {
	forwarder.MechanismsMonitorServer
	Init(*ForwarderConfig) error
	CreateForwarderServer(*ForwarderConfig) forwarder.ForwarderServer
}

// TODO Convert all the defaults to properly use NsmBaseDir
const (
	NSMBaseDirKey                        = "NSM_BASEDIR"
	NSMBaseDirDefault                    = "/var/lib/networkservicemesh/"
	ForwarderRegistrarSocketKey          = "FORWARDER_REGISTRAR_SOCKET"
	ForwarderRegistrarSocketDefault      = "/var/lib/networkservicemesh/nsm.forwarder-registrar.io.sock"
	ForwarderRegistrarSocketTypeKey      = "FORWARDER_REGISTRAR_SOCKET_TYPE"
	ForwarderRegistrarSocketTypeDefault  = "unix"
	ForwarderMetricsEnabledKey           = "METRICS_COLLECTOR_ENABLED"
	ForwarderMetricsEnabledDefault       = false
	ForwarderMetricsRequestPeriodKey     = "METRICS_COLLECTOR_REQUEST_PERIOD"
	ForwarderMetricsRequestPeriodDefault = time.Second * 2
	ForwarderNameKey                     = "FORWARDER_NAME"
	ForwarderNameDefault                 = "vppagent"
	ForwarderSocketKey                   = "FORWARDER_SOCKET"
	ForwarderSocketDefault               = "/var/lib/networkservicemesh/nsm-vppagent.forwarder.sock"
	ForwarderSocketTypeKey               = "FORWARDER_SOCKET_TYPE"
	ForwarderSocketTypeDefault           = "unix"
	ForwarderSrcIPKey                    = "NSM_FORWARDER_SRC_IP"
)

// ForwarderConfig keeps the common configuration for a forwarding plane
type ForwarderConfig struct {
	Name                    string
	NSMBaseDir              string
	RegistrarSocket         string
	RegistrarSocketType     string
	ForwarderSocket         string
	ForwarderSocketType     string
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

func createForwarderConfig(forwarderGoals *ForwarderProbeGoals) *ForwarderConfig {
	cfg := &ForwarderConfig{}
	var ok bool

	cfg.Name, ok = os.LookupEnv(ForwarderNameKey)
	if !ok {
		logrus.Debugf("%s not set, using default %s", ForwarderNameKey, ForwarderNameDefault)
		cfg.Name = ForwarderNameDefault
	}

	cfg.ForwarderSocket, ok = os.LookupEnv(ForwarderSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", ForwarderSocketKey, ForwarderSocketDefault)
		cfg.ForwarderSocket = ForwarderSocketDefault
	}
	logrus.Infof("ForwarderSocket: %s", cfg.ForwarderSocket)

	err := tools.SocketCleanup(cfg.ForwarderSocket)
	if err != nil {
		logrus.Fatalf("Error cleaning up socket %s: %s", cfg.ForwarderSocket, err)
	} else {
		forwarderGoals.SetSocketCleanReady()
	}

	cfg.ForwarderSocketType, ok = os.LookupEnv(ForwarderSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", ForwarderSocketTypeKey, ForwarderSocketTypeDefault)
		cfg.ForwarderSocketType = ForwarderSocketTypeDefault
	}
	logrus.Infof("ForwarderSocketType: %s", cfg.ForwarderSocketType)

	cfg.NSMBaseDir, ok = os.LookupEnv(NSMBaseDirKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", NSMBaseDirKey, NSMBaseDirDefault)
		cfg.NSMBaseDir = NSMBaseDirDefault
	}
	logrus.Infof("NSMBaseDir: %s", cfg.NSMBaseDir)

	cfg.RegistrarSocket, ok = os.LookupEnv(ForwarderRegistrarSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", ForwarderRegistrarSocketKey, ForwarderRegistrarSocketDefault)
		cfg.RegistrarSocket = ForwarderRegistrarSocketDefault
	}
	logrus.Infof("RegistrarSocket: %s", cfg.RegistrarSocket)

	cfg.RegistrarSocketType, ok = os.LookupEnv(ForwarderRegistrarSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", ForwarderRegistrarSocketTypeKey, ForwarderRegistrarSocketTypeDefault)
		cfg.RegistrarSocketType = ForwarderRegistrarSocketTypeDefault
	}
	logrus.Infof("RegistrarSocketType: %s", cfg.RegistrarSocketType)

	cfg.GRPCserver = tools.NewServer(context.Background())

	cfg.Monitor = monitor_crossconnect.NewMonitorServer()
	crossconnect.RegisterMonitorCrossConnectServer(cfg.GRPCserver, cfg.Monitor)

	cfg.MetricsEnabled = ForwarderMetricsEnabledDefault
	val, ok := os.LookupEnv(ForwarderMetricsEnabledKey)
	if ok {
		res, err := strconv.ParseBool(val)
		if err == nil {
			cfg.MetricsEnabled = res
		}
	}
	logrus.Infof("MetricsEnabled: %v", cfg.MetricsEnabled)

	if cfg.MetricsEnabled {
		cfg.MetricsPeriod = ForwarderMetricsRequestPeriodDefault
		if val, ok = os.LookupEnv(ForwarderMetricsRequestPeriodKey); ok {
			parsedPeriod, err := time.ParseDuration(val)
			if err == nil {
				cfg.MetricsPeriod = parsedPeriod
			}
		}
		logrus.Infof("MetricsPeriod: %v ", cfg.MetricsPeriod)
	}

	srcIPStr, ok := os.LookupEnv(ForwarderSrcIPKey)
	if !ok {
		logrus.Fatalf("Env variable %s must be set to valid srcIP for use for tunnels from this Pod.  Consider using downward API to do so.", ForwarderSrcIPKey)
	} else {
		forwarderGoals.SetSrcIPReady()
	}
	cfg.SrcIP = net.ParseIP(srcIPStr)
	if cfg.SrcIP == nil {
		logrus.Fatalf("Env variable %s must be set to a valid IP address, was set to %s", ForwarderSrcIPKey, srcIPStr)
	} else {
		forwarderGoals.SetValidIPReady()
	}
	cfg.EgressInterface, err = NewEgressInterface(cfg.SrcIP)
	if err != nil {
		logrus.Fatalf("Unable to find egress Interface: %s", err)
	} else {
		forwarderGoals.SetNewEgressIFReady()
	}
	logrus.Infof("SrcIP: %s, IfaceName: %s, SrcIPNet: %s", cfg.SrcIP, cfg.EgressInterface.Name(), cfg.EgressInterface.SrcIPNet())

	return cfg
}

// CreateForwarder creates new Forwarder Registrar client
func CreateForwarder(dp NSMForwarder, forwarderGoals *ForwarderProbeGoals) *ForwarderRegistration {
	start := time.Now()
	// Populate common configuration
	config := createForwarderConfig(forwarderGoals)

	// Initialize the forwarder
	err := dp.Init(config)
	if err != nil {
		logrus.Fatalf("Forwarder initialization failed: %s ", err)
	}

	// Verify the configuration is populated
	if !sanityCheckConfig(config) {
		logrus.Fatalf("Forwarder configuration sanity check failed: %s ", err)
	}

	// Prepare the gRPC server
	config.Listener, err = net.Listen(config.ForwarderSocketType, config.ForwarderSocket)
	if err != nil {
		logrus.Fatalf("Error listening on socket %s: %s ", config.ForwarderSocket, err)
	} else {
		forwarderGoals.SetSocketListenReady()
	}

	forwarder.RegisterForwarderServer(config.GRPCserver, dp.CreateForwarderServer(config))
	forwarder.RegisterMechanismsMonitorServer(config.GRPCserver, dp)

	// Start the server
	logrus.Infof("Creating %s server...", config.Name)
	go func() {
		_ = config.GRPCserver.Serve(config.Listener)
	}()
	logrus.Infof("%s server serving", config.Name)

	logrus.Debugf("Starting the %s forwarder server took: %s", config.Name, time.Since(start))

	logrus.Info("Creating Forwarder Registrar Client...")
	registrar := NewForwarderRegistrarClient(config.RegistrarSocketType, config.RegistrarSocket)
	registration := registrar.Register(context.Background(), config.Name, config.ForwarderSocket, nil, nil)
	logrus.Info("Registered Forwarder Registrar Client")

	return registration
}

func sanityCheckConfig(forwarderConfig *ForwarderConfig) bool {
	return len(forwarderConfig.Name) > 0 &&
		len(forwarderConfig.NSMBaseDir) > 0 &&
		len(forwarderConfig.RegistrarSocket) > 0 &&
		len(forwarderConfig.RegistrarSocketType) > 0 &&
		len(forwarderConfig.ForwarderSocket) > 0 &&
		len(forwarderConfig.ForwarderSocketType) > 0
}

// SanityCheckConnectionType checks whether the forwarding plane supports the connection type in the request
func SanityCheckConnectionType(mechanisms *Mechanisms, crossConnect *crossconnect.CrossConnect) error {
	localFound, remoteFound := false, false
	/* Verify local mechanisms */
	for _, mech := range mechanisms.LocalMechanisms {
		if compat.ConnectionUnifiedToLocal(crossConnect.GetLocalSource()).GetMechanism().GetType() == mech.GetType() ||
			compat.ConnectionUnifiedToLocal(crossConnect.GetLocalDestination()).GetMechanism().GetType() == mech.GetType() {
			localFound = true
			break
		}
	}
	/* Verify remote mechanisms */
	for _, mech := range mechanisms.RemoteMechanisms {
		if compat.ConnectionUnifiedToRemote(crossConnect.GetRemoteSource()).GetMechanism().GetType() == mech.GetType() ||
			compat.ConnectionUnifiedToRemote(crossConnect.GetRemoteDestination()).GetMechanism().GetType() == mech.GetType() {
			remoteFound = true
			break
		}
	}
	/* If none of them matched, mechanism is not supported by the forwarding plane */
	if !localFound && !remoteFound {
		return errors.New("connection mechanism type not supported by the forwarding plane")
	}
	return nil
}
