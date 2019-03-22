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
	Init(*DataplaneConfig) error
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

type DataplaneConfig struct {
	BaseCfg    *DataplaneConfigBase
	GRPCserver *grpc.Server
	Monitor    *crossconnect_monitor.CrossConnectMonitor
	Listener   net.Listener
}

func createDataplaneConfig() *DataplaneConfig {
	dpConfig := &DataplaneConfig{
		BaseCfg: &DataplaneConfigBase{},
	}
	var ok bool

	dpConfig.BaseCfg.NSMBaseDir, ok = os.LookupEnv(NSMBaseDirKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", NSMBaseDirKey, NSMBaseDirDefault)
		dpConfig.BaseCfg.NSMBaseDir = NSMBaseDirDefault
	}
	logrus.Infof("NSMBaseDir: %s", dpConfig.BaseCfg.NSMBaseDir)

	dpConfig.BaseCfg.RegistrarSocket, ok = os.LookupEnv(DataplaneRegistrarSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketKey, DataplaneRegistrarSocketDefault)
		dpConfig.BaseCfg.RegistrarSocket = DataplaneRegistrarSocketDefault
	}
	logrus.Infof("RegistrarSocket: %s", dpConfig.BaseCfg.RegistrarSocket)

	dpConfig.BaseCfg.RegistrarSocketType, ok = os.LookupEnv(DataplaneRegistrarSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneRegistrarSocketTypeKey, DataplaneRegistrarSocketTypeDefault)
		dpConfig.BaseCfg.RegistrarSocketType = DataplaneRegistrarSocketTypeDefault
	}
	logrus.Infof("RegistrarSocketType: %s", dpConfig.BaseCfg.RegistrarSocketType)

	tracer := opentracing.GlobalTracer()
	dpConfig.GRPCserver = grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	dpConfig.Monitor = crossconnect_monitor.NewCrossConnectMonitor()
	crossconnect.RegisterMonitorCrossConnectServer(dpConfig.GRPCserver, dpConfig.Monitor)

	monitor_crossconnect_server.NewMonitorNetNsInodeServer(dpConfig.Monitor)

	return dpConfig
}

func CreateDataplane(dp NSMDataplane) *dataplaneRegistration {
	start := time.Now()
	// Populate common configuration
	cfg := createDataplaneConfig()

	// Initialize the dataplane
	err := dp.Init(cfg)

	if err != nil {
		logrus.Fatalf("Dataplane initialization failed: %s ", err)
	}

	// Register the gRPC server
	dataplane.RegisterDataplaneServer(cfg.GRPCserver, dp)

	// Start the server
	logrus.Infof("Creating %s server...", cfg.BaseCfg.Name)
	go cfg.GRPCserver.Serve(cfg.Listener)
	logrus.Infof("%s server serving", cfg.BaseCfg.Name)

	logrus.Debugf("Starting the %s dataplane server took: %s", cfg.BaseCfg.Name, time.Since(start))

	logrus.Info("Creating Dataplane Registrar Client...")
	registrar := NewDataplaneRegistrarClient(cfg.BaseCfg.RegistrarSocketType, cfg.BaseCfg.RegistrarSocket)
	registration := registrar.Register(context.Background(), cfg.BaseCfg.Name, cfg.BaseCfg.DataplaneSocket, nil, nil)
	logrus.Info("Registered Dataplane Registrar Client")

	return registration
}
