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

package kernelforwarder

import (
	"context"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"

	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

// VPPAgent related constants
const (
	DataplaneMetricsCollectorEnabledKey           = "METRICS_COLLECTOR_ENABLED"
	DataplaneMetricsCollectorRequestPeriodKey     = "METRICS_COLLECTOR_REQUEST_PERIOD"
	DataplaneMetricsCollectorRequestPeriodDefault = time.Second * 2
	DataplaneNameKey                              = "DATAPLANE_NAME"
	DataplaneNameDefault                          = "vppagent"
	DataplaneSocketKey                            = "DATAPLANE_SOCKET"
	DataplaneSocketDefault                        = "/var/lib/networkservicemesh/nsm-vppagent.dataplane.sock"
	DataplaneSocketTypeKey                        = "DATAPLANE_SOCKET_TYPE"
	DataplaneSocketTypeDefault                    = "unix"
	DataplaneEndpointKey                          = "VPPAGENT_ENDPOINT"
	DataplaneEndpointDefault                      = "localhost:9111"
	SrcIPEnvKey                                   = "NSM_DATAPLANE_SRC_IP"
	ManagementInterface                           = "mgmt"
)

type KernelForwarder struct {
	vppAgentEndpoint     string
	common               *common.DataplaneConfigBase
	metricsCollector     *MetricsCollector
	mechanisms           *Mechanisms
	updateCh             chan *Mechanisms
	srcIP                net.IP
	egressInterface      common.EgressInterface
	monitor              monitor_crossconnect.MonitorServer
}

func CreateKernelForwarder() *KernelForwarder {
	return &KernelForwarder{}
}

// Mechanisms is a message used to communicate any changes in operational parameters and constraints
type Mechanisms struct {
	remoteMechanisms []*remote.Mechanism
	localMechanisms  []*local.Mechanism
}

func (v *KernelForwarder) MonitorMechanisms(empty *empty.Empty, updateSrv dataplane.Dataplane_MonitorMechanismsServer) error {
	logrus.Infof("MonitorMechanisms was called")
	initialUpdate := &dataplane.MechanismUpdate{
		RemoteMechanisms: v.mechanisms.remoteMechanisms,
		LocalMechanisms:  v.mechanisms.localMechanisms,
	}
	logrus.Infof("Sending MonitorMechanisms update: %v", initialUpdate)
	if err := updateSrv.Send(initialUpdate); err != nil {
		logrus.Errorf("vpp-agent dataplane server: Detected error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
		return nil
	}
	for {
		select {
		// Waiting for any updates which might occur during a life of dataplane module and communicating
		// them back to NSM.
		case update := <-v.updateCh:
			v.mechanisms = update
			logrus.Infof("Sending MonitorMechanisms update: %v", update)
			if err := updateSrv.Send(&dataplane.MechanismUpdate{
				RemoteMechanisms: update.remoteMechanisms,
				LocalMechanisms:  update.localMechanisms,
			}); err != nil {
				logrus.Errorf("vpp dataplane server: Detected error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
				return nil
			}
		}
	}
}

func (v *KernelForwarder) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	logrus.Infof("Request() called with %v", crossConnect)
	xcon, err := v.ConnectOrDisConnect(ctx, crossConnect, true)
	if err != nil {
		return nil, err
	}
	v.monitor.Update(xcon)
	logrus.Infof("Request() called with %v returning: %v", crossConnect, xcon)
	return xcon, err
}

func (v *KernelForwarder) handleKernelConnectionLocal(crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	/* Create a connection */
	if connect {
		/* 1. Get the connection configuration */
		cfg, err := getConnectionConfig(crossConnect)
		if err != nil {
			logrus.Errorf("Failed to get the configuration for local connection - %v", err)
			return crossConnect, err
		}
		/* 2. Get namespace handlers from their path - source and destination */
		srcNsHandle, err := netns.GetFromPath(cfg.srcNsPath)
		defer srcNsHandle.Close()
		if err != nil {
			logrus.Errorf("Failed to get source namespace handler from path - %v", err)
			return crossConnect, err
		}
		dstNsHandle, err := netns.GetFromPath(cfg.dstNsPath)
		defer dstNsHandle.Close()
		if err != nil {
			logrus.Errorf("Failed to get destination namespace handler from path - %v", err)
			return crossConnect, err
		}
		/* 3. Create a VETH pair and inject each end in the corresponding namespace */
		if err = createVETH(cfg, srcNsHandle, dstNsHandle); err != nil {
			logrus.Errorf("Failed to create the VETH pair - %v", err)
			return crossConnect, err
		}
		/* 4. Bring up and configure each pair end with its IP address */
		setupVETHEnd(srcNsHandle, cfg.srcName, cfg.srcIP)
		setupVETHEnd(dstNsHandle, cfg.dstName, cfg.dstIP)
	}
	/* Delete a connection */
	return crossConnect, nil
}

func (v *KernelForwarder) handleKernelConnectionRemote(ctx context.Context, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	logrus.Errorf("Remote connection is not supported yet.")
	return crossConnect, nil
}

func (v *KernelForwarder) ConnectOrDisConnect(ctx context.Context, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	/* 1. Handle local connection */
	if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE {
		return v.handleKernelConnectionLocal(crossConnect, connect)
	}
	/* 2. Handle remote connection */
	return v.handleKernelConnectionRemote(ctx, crossConnect, connect)
}

func (v *KernelForwarder) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	logrus.Infof("Close() called with %#v", crossConnect)
	xcon, err := v.ConnectOrDisConnect(ctx, crossConnect, false)
	if err != nil {
		logrus.Warn(err)
	}
	v.monitor.Delete(xcon)
	return &empty.Empty{}, nil
}

// Init makes setup for the KernelForwarder
func (v *KernelForwarder) Init(common *common.DataplaneConfigBase, monitor monitor_crossconnect.MonitorServer) error {
	tracer, closer := tools.InitJaeger("kernel-forwarder")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	v.common = common
	v.setDataplaneConfigBase()
	v.setDataplaneConfigVPPAgent(monitor)
	v.setupMetricsCollector(monitor)
	return nil
}

func (v *KernelForwarder) setupMetricsCollector(monitor metrics.MetricsMonitor) {
	val, ok := os.LookupEnv(DataplaneMetricsCollectorEnabledKey)
	if ok {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			logrus.Errorf("Metrics collector using default value for %v, %v ", DataplaneMetricsCollectorEnabledKey, err)
		} else if !enabled {
			logrus.Info("Metics collector is disabled")
			return
		}
	}
	requestPeriod := DataplaneMetricsCollectorRequestPeriodDefault
	if val, ok = os.LookupEnv(DataplaneMetricsCollectorRequestPeriodKey); ok {
		parsedPeriod, err := time.ParseDuration(val)
		if err != nil {
			logrus.Errorf("Metrics collector using default request period, %v ", err)
		} else {
			requestPeriod = parsedPeriod
		}
	}
	logrus.Infof("Metrics collector request period: %v ", requestPeriod)
	v.metricsCollector = NewMetricsCollector(requestPeriod)
	v.metricsCollector.CollectAsync(monitor, v.vppAgentEndpoint)
}

func (v *KernelForwarder) setDataplaneConfigBase() {
	var ok bool

	v.common.Name, ok = os.LookupEnv(DataplaneNameKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneNameKey, DataplaneNameDefault)
		v.common.Name = DataplaneNameDefault
	}

	logrus.Infof("Starting dataplane - %s", v.common.Name)
	v.common.DataplaneSocket, ok = os.LookupEnv(DataplaneSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketKey, DataplaneSocketDefault)
		v.common.DataplaneSocket = DataplaneSocketDefault
	}
	logrus.Infof("DataplaneSocket: %s", v.common.DataplaneSocket)

	v.common.DataplaneSocketType, ok = os.LookupEnv(DataplaneSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketTypeKey, DataplaneSocketTypeDefault)
		v.common.DataplaneSocketType = DataplaneSocketTypeDefault
	}
	logrus.Infof("DataplaneSocketType: %s", v.common.DataplaneSocketType)
}

func (v *KernelForwarder) setDataplaneConfigVPPAgent(monitor monitor_crossconnect.MonitorServer) {
	var err error

	v.monitor = monitor

	srcIPStr, ok := os.LookupEnv(SrcIPEnvKey)
	if !ok {
		logrus.Fatalf("Env variable %s must be set to valid srcIP for use for tunnels from this Pod.  Consider using downward API to do so.", SrcIPEnvKey)
		common.SetSrcIPFailed()
	}
	v.srcIP = net.ParseIP(srcIPStr)
	if v.srcIP == nil {
		logrus.Fatalf("Env variable %s must be set to a valid IP address, was set to %s", SrcIPEnvKey, srcIPStr)
		common.SetValidIPFailed()
	}
	v.egressInterface, err = common.NewEgressInterface(v.srcIP)
	if err != nil {
		logrus.Fatalf("Unable to find egress Interface: %s", err)
		common.SetNewEgressIFFailed()
	}
	logrus.Infof("SrcIP: %s, IfaceName: %s, SrcIPNet: %s", v.srcIP, v.egressInterface.Name(), v.egressInterface.SrcIPNet())

	err = tools.SocketCleanup(v.common.DataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error cleaning up socket %s: %s", v.common.DataplaneSocket, err)
		common.SetSocketCleanFailed()
	}

	v.vppAgentEndpoint, ok = os.LookupEnv(DataplaneEndpointKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneEndpointKey, DataplaneEndpointDefault)
		v.vppAgentEndpoint = DataplaneEndpointDefault
	}
	logrus.Infof("vppAgentEndpoint: %s", v.vppAgentEndpoint)

	v.updateCh = make(chan *Mechanisms, 1)
	v.mechanisms = &Mechanisms{
		localMechanisms: []*local.Mechanism{
			{
				Type: local.MechanismType_MEM_INTERFACE,
			},
			{
				Type: local.MechanismType_KERNEL_INTERFACE,
			},
		},
		remoteMechanisms: []*remote.Mechanism{
			{
				Type: remote.MechanismType_VXLAN,
				Parameters: map[string]string{
					remote.VXLANSrcIP: v.egressInterface.SrcIPNet().IP.String(),
				},
			},
		},
	}
}
