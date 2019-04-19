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

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type NSMDataplane interface {
	dataplane.DataplaneServer
	Init(*DataplaneConfigBase, *crossconnect_monitor.CrossConnectMonitor) error
}

// TODO Convert all the defaults to properly use NsmBaseDir
const (
	NSMBaseDirKey                       = "NSM_BASEDIR"
	NSMBaseDirDefault                   = "/var/lib/networkservicemesh/"
	DataplaneRegistrarSocketKey         = "DATAPLANE_REGISTRAR_SOCKET"
	DataplaneRegistrarSocketDefault     = "/var/lib/networkservicemesh/nsm.dataplane-registrar.io.sock"
	DataplaneRegistrarSocketTypeKey     = "DATAPLANE_REGISTRAR_SOCKET_TYPE"
	DataplaneRegistrarSocketTypeDefault = "unix"
)

type DataplaneConfigBase struct {
	Name                string
	NSMBaseDir          string
	RegistrarSocket     string
	RegistrarSocketType string
	DataplaneSocket     string
	DataplaneSocketType string
}

type dataplaneConfig struct {
	common     *DataplaneConfigBase
	gRPCserver *grpc.Server
	monitor    *crossconnect_monitor.CrossConnectMonitor
	listener   net.Listener
}

func createDataplaneConfig() *dataplaneConfig {
	dpConfig := &dataplaneConfig{
		common: &DataplaneConfigBase{},
	}
	var ok bool

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

	dpConfig.monitor = crossconnect_monitor.NewCrossConnectMonitor()
	crossconnect.RegisterMonitorCrossConnectServer(dpConfig.gRPCserver, dpConfig.monitor)
	monitor_crossconnect_server.NewMonitorNetNsInodeServer(dpConfig.monitor)

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
