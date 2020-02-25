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

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
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
	RemoteMechanisms []*networkservice.Mechanism
	LocalMechanisms  []*networkservice.Mechanism
}

func getEnvWithDefault(span spanhelper.SpanHelper, env, defaultValue string) string {
	result, ok := os.LookupEnv(env)
	if !ok {
		span.LogObject(env, fmt.Sprintf("%s (default value)", defaultValue))
		result = defaultValue
	} else {
		span.LogObject(env, result)
	}
	return result
}

func getEnvWithDefaultBool(span spanhelper.SpanHelper, env string, defaultValue bool) bool {
	result := defaultValue
	resultVal, ok := os.LookupEnv(env)
	var err error
	if !ok {
		span.LogObject(env, fmt.Sprintf("%v (default value)", defaultValue))
		result = defaultValue
	} else {
		span.LogObject(env, result)
		result, err = strconv.ParseBool(resultVal)
		span.LogError(err)
		if err != nil {
			result = defaultValue
		}
	}
	return result
}

func createForwarderConfig(ctx context.Context, forwarderGoals *ForwarderProbeGoals) *ForwarderConfig {
	span := spanhelper.FromContext(ctx, "createForwartderConfig")
	defer span.Finish()

	cfg := &ForwarderConfig{}

	cfg.Name = getEnvWithDefault(span, ForwarderNameKey, ForwarderNameDefault)
	cfg.ForwarderSocket = getEnvWithDefault(span, ForwarderSocketKey, ForwarderSocketDefault)

	if err := tools.SocketCleanup(cfg.ForwarderSocket); err != nil {
		span.Logger().Fatalf("Error cleaning up socket %s: %s", cfg.ForwarderSocket, err)
	} else {
		forwarderGoals.SetSocketCleanReady()
	}

	cfg.ForwarderSocketType = getEnvWithDefault(span, ForwarderSocketTypeKey, ForwarderSocketTypeDefault)

	cfg.NSMBaseDir = getEnvWithDefault(span, NSMBaseDirKey, NSMBaseDirDefault)
	cfg.RegistrarSocket = getEnvWithDefault(span, ForwarderRegistrarSocketKey, ForwarderRegistrarSocketDefault)
	cfg.RegistrarSocketType = getEnvWithDefault(span, ForwarderRegistrarSocketTypeKey, ForwarderRegistrarSocketTypeDefault)

	cfg.MetricsEnabled = getEnvWithDefaultBool(span, ForwarderMetricsEnabledKey, ForwarderMetricsEnabledDefault)
	if cfg.MetricsEnabled {
		cfg.MetricsPeriod = ForwarderMetricsRequestPeriodDefault

		if val, ok := os.LookupEnv(ForwarderMetricsRequestPeriodKey); ok {
			parsedPeriod, err := time.ParseDuration(val)
			if err == nil {
				cfg.MetricsPeriod = parsedPeriod
			}
		}
		span.Logger().Infof("MetricsPeriod: %v ", cfg.MetricsPeriod)
	}

	srcIPStr, ok := os.LookupEnv(ForwarderSrcIPKey)
	if !ok {
		span.Logger().Fatalf("Env variable %s must be set to valid srcIP for use for tunnels from this Pod.  Consider using downward API to do so.", ForwarderSrcIPKey)
	} else {
		forwarderGoals.SetSrcIPReady()
	}
	cfg.SrcIP = net.ParseIP(srcIPStr)
	if cfg.SrcIP == nil {
		span.Logger().Fatalf("Env variable %s must be set to a valid IP address, was set to %s", ForwarderSrcIPKey, srcIPStr)
	} else {
		forwarderGoals.SetValidIPReady()
	}
	var err error
	cfg.EgressInterface, err = NewEgressInterface(cfg.SrcIP)
	if err != nil {
		span.Logger().Fatalf("Unable to find egress Interface: %s", err)
	} else {
		forwarderGoals.SetNewEgressIFReady()
	}
	span.Logger().Infof("SrcIP: %s, IfaceName: %s, SrcIPNet: %s", cfg.SrcIP, cfg.EgressInterface.Name(), cfg.EgressInterface.SrcIPNet())
	span.LogObject("config", cfg)
	return cfg
}

// CreateForwarder creates new Forwarder Registrar client
func CreateForwarder(ctx context.Context, dp NSMForwarder, forwarderGoals *ForwarderProbeGoals) *ForwarderRegistration {
	span := spanhelper.FromContext(ctx, "CreateForwarder")
	defer span.Finish()
	// Populate common configuration
	config := createForwarderConfig(span.Context(), forwarderGoals)

	// Initialize GRPC server
	config.GRPCserver = tools.NewServer(span.Context())
	config.Monitor = monitor_crossconnect.NewMonitorServer()
	crossconnect.RegisterMonitorCrossConnectServer(config.GRPCserver, config.Monitor)

	// Initialize the forwarder
	err := dp.Init(config)
	if err != nil {
		span.Logger().Fatalf("Forwarder initialization failed: %s ", err)
	}

	// Verify the configuration is populated
	if !sanityCheckConfig(config) {
		span.Logger().Fatalf("Forwarder configuration sanity check failed: %s ", err)
	}

	// Prepare the gRPC server
	config.Listener, err = net.Listen(config.ForwarderSocketType, config.ForwarderSocket)
	if err != nil {
		span.Logger().Fatalf("Error listening on socket %s: %s ", config.ForwarderSocket, err)
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
	span.Logger().Infof("%s server serving", config.Name)
	span.Logger().Info("Creating Forwarder Registrar Client...")
	registrar := NewForwarderRegistrarClient(config.RegistrarSocketType, config.RegistrarSocket)
	registration := registrar.Register(span.Context(), config.Name, config.ForwarderSocket, nil, nil)
	span.Logger().Info("Registered Forwarder Registrar Client")

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
		if crossConnect.GetLocalSource().GetMechanism().GetType() == mech.GetType() ||
			crossConnect.GetLocalDestination().GetMechanism().GetType() == mech.GetType() {
			localFound = true
			break
		}
	}
	/* Verify remote mechanisms */
	for _, mech := range mechanisms.RemoteMechanisms {
		if crossConnect.GetRemoteSource().GetMechanism().GetType() == mech.GetType() ||
			crossConnect.GetRemoteDestination().GetMechanism().GetType() == mech.GetType() {
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
