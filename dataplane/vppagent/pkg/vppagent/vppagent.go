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

package vppagent

import (
	"context"
	"os"
	"time"

	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/ligato/vpp-agent/plugins/vpp/model/acl"
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/converter"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/memif"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type VPPAgent struct {
	// Parameters set in constructor
	vppAgentEndpoint string
	monitor          *crossconnect_monitor.CrossConnectMonitor

	// Internal state from here on
	mechanisms           *Mechanisms
	updateCh             chan *Mechanisms
	baseDir              string
	egressInterface      *common.EgressInterface
	directMemifConnector *memif.DirectMemifConnector
}

func NewVPPAgent(monitor *crossconnect_monitor.CrossConnectMonitor, baseDir string, egressInterface *common.EgressInterface) *VPPAgent {

	vppAgentEndpoint, ok := os.LookupEnv(common.DataplaneVPPAgentEndpointKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", common.DataplaneVPPAgentEndpointKey, common.DefaultVPPAgentEndpoint)
		vppAgentEndpoint = common.DefaultVPPAgentEndpoint
	}
	logrus.Infof("vppAgentEndpoint: %s", vppAgentEndpoint)

	// TODO provide some validations here for inputs
	rv := &VPPAgent{
		updateCh:         make(chan *Mechanisms, 1),
		vppAgentEndpoint: vppAgentEndpoint,
		baseDir:          baseDir,
		egressInterface:  egressInterface,
		monitor:          monitor,
		mechanisms: &Mechanisms{
			localMechanisms: []*local.Mechanism{
				{
					Type: local.MechanismType_KERNEL_INTERFACE,
				},
				{
					Type: local.MechanismType_MEM_INTERFACE,
				},
			},
			remoteMechanisms: []*remote.Mechanism{
				{
					Type: remote.MechanismType_VXLAN,
					Parameters: map[string]string{
						remote.VXLANSrcIP: egressInterface.SrcIPNet().IP.String(),
					},
				},
			},
		},
		directMemifConnector: memif.NewDirectMemifConnector(baseDir),
	}
	rv.reset()
	rv.programMgmtInterface()
	return rv
}

// Mechanisms is a message used to communicate any changes in operational parameters and constraints
type Mechanisms struct {
	remoteMechanisms []*remote.Mechanism
	localMechanisms  []*local.Mechanism
}

func (v *VPPAgent) MonitorMechanisms(empty *empty.Empty, updateSrv dataplane.Dataplane_MonitorMechanismsServer) error {
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
				logrus.Errorf("vpp dataplane server: Deteced error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
				return nil
			}
		}
	}
}

func (v *VPPAgent) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	logrus.Infof("Request(ConnectRequest) called with %v", crossConnect)
	xcon, err := v.ConnectOrDisConnect(ctx, crossConnect, true)
	v.monitor.Update(xcon)
	logrus.Infof("Request(ConnectRequest) called with %v returning: %v", crossConnect, xcon)
	return xcon, err
}

func (v *VPPAgent) ConnectOrDisConnect(ctx context.Context, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_MEM_INTERFACE &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_MEM_INTERFACE {
		return v.directMemifConnector.ConnectOrDisConnect(crossConnect, connect)
	}

	// TODO look at whether keepin a single conn might be better
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(v.vppAgentEndpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return nil, err
	}
	defer conn.Close()
	client := rpc.NewDataChangeServiceClient(conn)
	conversionParameters := &converter.CrossConnectConversionParameters{
		BaseDir: v.baseDir,
	}
	dataChange, err := converter.NewCrossConnectConverter(crossConnect, conversionParameters).ToDataRequest(nil, connect)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logrus.Infof("Sending DataChange to vppagent: %v", dataChange)
	if connect {
		_, err = client.Put(ctx, dataChange)
	} else {
		_, err = client.Del(ctx, dataChange)
	}
	if err != nil {
		logrus.Error(err)
		// TODO handle connection tracking
		// TODO handle teardown of any partial config that happened
		return crossConnect, err
	}
	return crossConnect, nil
}

func (v *VPPAgent) reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	tools.WaitForPortAvailable(ctx, "tcp", v.vppAgentEndpoint, 100*time.Millisecond)
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(v.vppAgentEndpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
	client := rpc.NewDataResyncServiceClient(conn)
	logrus.Infof("Resetting vppagent...")
	_, err = client.Resync(context.Background(), &rpc.DataRequest{})
	if err != nil {
		logrus.Errorf("failed to reset vppagent: %s", err)
	}
	logrus.Infof("Finished resetting vppagent...")
	return nil
}

func (v *VPPAgent) programMgmtInterface() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	tools.WaitForPortAvailable(ctx, "tcp", v.vppAgentEndpoint, 100*time.Millisecond)
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(v.vppAgentEndpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
	client := rpc.NewDataChangeServiceClient(conn)
	dataRequest := &rpc.DataRequest{
		Interfaces: []*interfaces.Interfaces_Interface{
			{
				Name:        "mgmt",
				Type:        interfaces.InterfaceType_AF_PACKET_INTERFACE,
				Enabled:     true,
				IpAddresses: []string{v.egressInterface.SrcIPNet().String()},
				PhysAddress: v.egressInterface.HardwareAddr.String(),
				Afpacket: &interfaces.Interfaces_Interface_Afpacket{
					HostIfName: v.egressInterface.Name,
				},
			},
		},
	}
	// When using AF_PACKET, both the kernel, and vpp receive the packets.
	// Since both vpp and the kernel have the same IP and hw address,
	// vpp will send icmp port unreachable messages out for anything
	// that is sent to that IP/mac address ... which screws up lots of things.
	// This causes vpp to have an ACL on the management interface such that
	// it drops anything that isn't destined for VXLAN (port 4789).
	// This way it avoids sending icmp port unreachable messages out.
	// This bug wasn't really obvious till we tried to switch to hostNetwork:true
	dataRequest.AccessLists = []*acl.AccessLists_Acl{
		&acl.AccessLists_Acl{
			AclName: "NSMmgmtInterfaceACL",
			Interfaces: &acl.AccessLists_Acl_Interfaces{
				Ingress: []string{dataRequest.Interfaces[0].Name},
			},
			Rules: []*acl.AccessLists_Acl_Rule{
				&acl.AccessLists_Acl_Rule{
					RuleName:  "NSMmgmtInterfaceACL permit VXLAN dst",
					AclAction: acl.AclAction_PERMIT,
					Match: &acl.AccessLists_Acl_Rule_Match{
						IpRule: &acl.AccessLists_Acl_Rule_Match_IpRule{
							Ip: &acl.AccessLists_Acl_Rule_Match_IpRule_Ip{
								DestinationNetwork: v.egressInterface.SrcIPNet().IP.String() + "/32",
								SourceNetwork:      "0.0.0.0/0",
							},
							Udp: &acl.AccessLists_Acl_Rule_Match_IpRule_Udp{
								DestinationPortRange: &acl.AccessLists_Acl_Rule_Match_IpRule_PortRange{
									LowerPort: 4789,
									UpperPort: 4789,
								},
								SourcePortRange: &acl.AccessLists_Acl_Rule_Match_IpRule_PortRange{
									LowerPort: 0,
									UpperPort: 65535,
								},
							},
						},
					},
				},
			},
		},
	}
	logrus.Infof("Setting up Mgmt Interface %v", dataRequest)
	_, err = client.Put(context.Background(), dataRequest)
	if err != nil {
		logrus.Errorf("Error Setting up Mgmt Interface: %s", err)
		return err
	}
	return nil
}

func (v *VPPAgent) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	logrus.Infof("vppagent.DisconnectRequest called with %#v", crossConnect)
	xcon, err := v.ConnectOrDisConnect(ctx, crossConnect, false)
	if err != nil {
		logrus.Warn(err)
	}
	v.monitor.Delete(xcon)
	return &empty.Empty{}, nil
}
