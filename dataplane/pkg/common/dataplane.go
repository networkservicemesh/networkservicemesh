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

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
)

type NSMDataplane interface {
	dataplane.DataplaneServer
	Init(*DataplaneConfigBase, monitor_crossconnect.MonitorServer) error
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
)

type DataplaneConfigBase struct {
	Name                string
	NSMBaseDir          string
	RegistrarSocket     string
	RegistrarSocketType string
	DataplaneSocket     string
	DataplaneSocketType string
	Mechanisms          *Mechanisms
	MetricsEnabled      bool
	MetricsPeriod       time.Duration
}

type dataplaneConfig struct {
	common     *DataplaneConfigBase
	gRPCserver *grpc.Server
	monitor    monitor_crossconnect.MonitorServer
	listener   net.Listener
}

// Mechanisms is a message used to communicate any changes in operational parameters and constraints
type Mechanisms struct {
	RemoteMechanisms []*remote.Mechanism
	LocalMechanisms  []*local.Mechanism
}

func createDataplaneConfig() *dataplaneConfig {
	dpConfig := &dataplaneConfig{
		common: &DataplaneConfigBase{},
	}
	var ok bool

	dpConfig.common.Name, ok = os.LookupEnv(DataplaneNameKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneNameKey, DataplaneNameDefault)
		dpConfig.common.Name = DataplaneNameDefault
	}
	logrus.Infof("Starting dataplane - %s", dpConfig.common.Name)

	dpConfig.common.DataplaneSocket, ok = os.LookupEnv(DataplaneSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketKey, DataplaneSocketDefault)
		dpConfig.common.DataplaneSocket = DataplaneSocketDefault
	}
	logrus.Infof("DataplaneSocket: %s", dpConfig.common.DataplaneSocket)

	dpConfig.common.DataplaneSocketType, ok = os.LookupEnv(DataplaneSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketTypeKey, DataplaneSocketTypeDefault)
		dpConfig.common.DataplaneSocketType = DataplaneSocketTypeDefault
	}
	logrus.Infof("DataplaneSocketType: %s", dpConfig.common.DataplaneSocketType)

	dpConfig.common.NSMBaseDir, ok = os.LookupEnv(NSMBaseDirKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", NSMBaseDirKey, NSMBaseDirDefault)
		dpConfig.common.NSMBaseDir = NSMBaseDirDefault
	}
	logrus.Infof("NSMBaseDir: %s", dpConfig.common.NSMBaseDir)

	dpConfig.common.RegistrarSocket, ok = os.LookupEnv(DataplaneRegistrarSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketKey, DataplaneRegistrarSocketDefault)
		dpConfig.common.RegistrarSocket = DataplaneRegistrarSocketDefault
	}
	logrus.Infof("RegistrarSocket: %s", dpConfig.common.RegistrarSocket)

	dpConfig.common.RegistrarSocketType, ok = os.LookupEnv(DataplaneRegistrarSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketTypeKey, DataplaneRegistrarSocketTypeDefault)
		dpConfig.common.RegistrarSocketType = DataplaneRegistrarSocketTypeDefault
	}
	logrus.Infof("RegistrarSocketType: %s", dpConfig.common.RegistrarSocketType)

	tracer := opentracing.GlobalTracer()
	dpConfig.gRPCserver = grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	dpConfig.monitor = monitor_crossconnect.NewMonitorServer()
	crossconnect.RegisterMonitorCrossConnectServer(dpConfig.gRPCserver, dpConfig.monitor)
	monitor_crossconnect_server.NewMonitorNetNsInodeServer(dpConfig.monitor)

	dpConfig.common.MetricsEnabled = DataplaneMetricsEnabledDefault
	val, ok := os.LookupEnv(DataplaneMetricsEnabledKey)
	if ok {
		res, err := strconv.ParseBool(val)
		if err == nil {
			dpConfig.common.MetricsEnabled = res
		}
	}
	logrus.Infof("MetricsEnabled: %v", dpConfig.common.MetricsEnabled)

	if dpConfig.common.MetricsEnabled {
		dpConfig.common.MetricsPeriod = DataplaneMetricsRequestPeriodDefault
		if val, ok = os.LookupEnv(DataplaneMetricsRequestPeriodKey); ok {
			parsedPeriod, err := time.ParseDuration(val)
			if err == nil {
				dpConfig.common.MetricsPeriod = parsedPeriod
			}
		}
		logrus.Infof("MetricsPeriod: %v ", dpConfig.common.MetricsPeriod)
	}
	return dpConfig
}

func CreateDataplane(dp NSMDataplane) *dataplaneRegistration {
	start := time.Now()
	// Populate common configuration
	config := createDataplaneConfig()

	// Initialize the dataplane
	err := dp.Init(config.common, config.monitor)
	if err != nil {
		logrus.Fatalf("Dataplane initialization failed: %s ", err)
	}

	// Verify the configuration is populated
	if !sanityCheckConfig(config.common) {
		logrus.Fatalf("Dataplane configuration sanity check failed: %s ", err)
	}

	// Prepare the gRPC server
	config.listener, err = net.Listen(config.common.DataplaneSocketType, config.common.DataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error listening on socket %s: %s ", config.common.DataplaneSocket, err)
		SetSocketListenFailed()
	}
	dataplane.RegisterDataplaneServer(config.gRPCserver, dp)

	// Start the server
	logrus.Infof("Creating %s server...", config.common.Name)
	go config.gRPCserver.Serve(config.listener)
	logrus.Infof("%s server serving", config.common.Name)

	logrus.Debugf("Starting the %s dataplane server took: %s", config.common.Name, time.Since(start))

	logrus.Info("Creating Dataplane Registrar Client...")
	registrar := NewDataplaneRegistrarClient(config.common.RegistrarSocketType, config.common.RegistrarSocket)
	registration := registrar.Register(context.Background(), config.common.Name, config.common.DataplaneSocket, nil, nil)
	logrus.Info("Registered Dataplane Registrar Client")

	return registration
}

func sanityCheckConfig(dataplaneConfig *DataplaneConfigBase) bool {
	return len(dataplaneConfig.Name) > 0 &&
		len(dataplaneConfig.NSMBaseDir) > 0 &&
		len(dataplaneConfig.RegistrarSocket) > 0 &&
		len(dataplaneConfig.RegistrarSocketType) > 0 &&
		len(dataplaneConfig.DataplaneSocket) > 0 &&
		len(dataplaneConfig.DataplaneSocketType) > 0
}
