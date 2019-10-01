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

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	vpp_acl "github.com/ligato/vpp-agent/api/models/vpp/acl"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	vpp_l3 "github.com/ligato/vpp-agent/api/models/vpp/l3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/dataplane/api/dataplane"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/converter"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/memif"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/vppagent/nsmonitor"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

// VPPAgent related constants
const (
	VPPEndpointKey      = "VPPAGENT_ENDPOINT"
	VPPEndpointDefault  = "localhost:9111"
	ManagementInterface = "mgmt"
)

type VPPAgent struct {
	vppAgentEndpoint     string
	metricsCollector     *MetricsCollector
	directMemifConnector *memif.DirectMemifConnector
	common               *common.DataplaneConfig
	ctx                  context.Context
}

// CreateVPPAgent - create VPP agent
func CreateVPPAgent(ctx context.Context) *VPPAgent {
	return &VPPAgent{
		ctx: ctx,
	}
}

// MonitorMechanisms sends mechanism updates
func (v *VPPAgent) MonitorMechanisms(empty *empty.Empty, updateSrv dataplane.MechanismsMonitor_MonitorMechanismsServer) error {
	span := spanhelper.FromContext(v.ctx, "MonitorMecnahisms")
	defer span.Finish()
	span.Logger().Infof("MonitorMechanisms was called")
	initialUpdate := &dataplane.MechanismUpdate{
		RemoteMechanisms: v.common.Mechanisms.RemoteMechanisms,
		LocalMechanisms:  v.common.Mechanisms.LocalMechanisms,
	}
	span.Logger().Infof("Sending MonitorMechanisms update: %v", initialUpdate)
	if err := updateSrv.Send(initialUpdate); err != nil {
		span.Logger().Errorf("vpp-agent dataplane server: Detected error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
		return nil
	}
	for {
		select {
		// Waiting for any updates which might occur during a life of dataplane module and communicating
		// them back to NSM.
		case update := <-v.common.MechanismsUpdateChannel:
			v.common.Mechanisms = update
			span.Logger().Infof("Sending MonitorMechanisms update: %v", update)
			if err := updateSrv.Send(&dataplane.MechanismUpdate{
				RemoteMechanisms: update.RemoteMechanisms,
				LocalMechanisms:  update.LocalMechanisms,
			}); err != nil {
				span.Logger().Errorf("vpp dataplane server: Detected error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
				return nil
			}
		}
	}
}

func (v *VPPAgent) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	logrus.Infof("Request(ConnectRequest) called with %v", crossConnect)
	xcon, err := v.connectOrDisconnect(ctx, crossConnect, true)
	if err != nil {
		return nil, err
	}
	v.common.Monitor.Update(ctx, xcon)
	logrus.Infof("Request(ConnectRequest) called with %v returning: %v", crossConnect, xcon)
	return xcon, err
}

func (v *VPPAgent) connectOrDisconnect(ctx context.Context, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_MEM_INTERFACE &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_MEM_INTERFACE {
		return v.directMemifConnector.ConnectOrDisconnect(crossConnect, connect)
	}

	// TODO look at whether keepin a single conn might be better
	conn, err := tools.DialTCPInsecure(v.vppAgentEndpoint)
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return nil, err
	}
	defer conn.Close()
	client := configurator.NewConfiguratorClient(conn)
	conversionParameters := &converter.CrossConnectConversionParameters{
		BaseDir: v.common.NSMBaseDir,
	}
	dataChange, err := converter.NewCrossConnectConverter(crossConnect, conversionParameters).ToDataRequest(nil, connect)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	if connect {
		entity := v.common.Monitor.Entities()[crossConnect.GetId()]
		if entity != nil {
			clearDataChange, cErr := converter.NewCrossConnectConverter(entity.(*crossconnect.CrossConnect), conversionParameters).MechanismsToDataRequest(nil, false)
			if cErr == nil && clearDataChange != nil {
				logrus.Infof("Sending clearing DataChange to vppagent: %v", proto.MarshalTextString(clearDataChange))
				_, cErr = client.Delete(ctx, &configurator.DeleteRequest{Delete: clearDataChange})
			}
			if cErr != nil {
				logrus.Warnf("Connection Mechanism was not cleared properly before updating: %s", cErr.Error())
			}
		}
	}

	logrus.Infof("Sending DataChange to vppagent: %v", proto.MarshalTextString(dataChange))
	if connect {
		_, err = client.Update(ctx, &configurator.UpdateRequest{Update: dataChange})
	} else {
		_, err = client.Delete(ctx, &configurator.DeleteRequest{Delete: dataChange})
	}

	v.printVppAgentConfiguration(client)

	if err != nil {
		logrus.Error(err)
		// TODO handle connection tracking
		// TODO handle teardown of any partial config that happened
		return crossConnect, err
	}
	return crossConnect, nil
}

func (v *VPPAgent) printVppAgentConfiguration(client configurator.ConfiguratorClient) {
	dumpResult, err := client.Dump(context.Background(), &configurator.DumpRequest{})
	if err != nil {
		logrus.Errorf("Failed to dump VPP-agent state %v", err)
	}
	logrus.Infof("VPP Agent Configuration: %v", proto.MarshalTextString(dumpResult))
}

func (v *VPPAgent) reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	tools.WaitForPortAvailable(ctx, "tcp", v.vppAgentEndpoint, 100*time.Millisecond)

	conn, err := tools.DialTCPInsecure(v.vppAgentEndpoint)
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
	client := configurator.NewConfiguratorClient(conn)
	logrus.Infof("Resetting vppagent...")
	_, err = client.Update(context.Background(), &configurator.UpdateRequest{Update: &configurator.Config{}, FullResync: true})
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

	conn, err := tools.DialTCPInsecure(v.vppAgentEndpoint)
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
	client := configurator.NewConfiguratorClient(conn)

	vppArpEntries := []*vpp.ARPEntry{}
	for _, arpEntry := range v.common.EgressInterface.ArpEntries() {
		vppArpEntries = append(vppArpEntries, &vpp.ARPEntry{
			Interface:   ManagementInterface,
			IpAddress:   arpEntry.IPAddress,
			PhysAddress: arpEntry.PhysAddress,
		})
	}

	dataRequest := &configurator.UpdateRequest{
		Update: &configurator.Config{
			VppConfig: &vpp.ConfigData{
				Interfaces: []*vpp.Interface{
					{
						Name:        ManagementInterface,
						Type:        vpp_interfaces.Interface_AF_PACKET,
						Enabled:     true,
						IpAddresses: []string{v.common.EgressInterface.SrcIPNet().String()},
						PhysAddress: v.common.EgressInterface.HardwareAddr().String(),
						Link: &vpp_interfaces.Interface_Afpacket{
							Afpacket: &vpp_interfaces.AfpacketLink{
								HostIfName: v.common.EgressInterface.Name(),
							},
						},
					},
				},
				// Add default route via default gateway
				Routes: []*vpp.Route{
					{
						Type:              vpp_l3.Route_INTER_VRF,
						OutgoingInterface: ManagementInterface,
						DstNetwork:        "0.0.0.0/0",
						Weight:            1,
						NextHopAddr:       v.common.EgressInterface.DefaultGateway().String(),
					},
				},
				// Add system arp entries
				Arps: vppArpEntries,
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
	dataRequest.Update.VppConfig.Acls = []*vpp.ACL{
		{
			Name: "NSMmgmtInterfaceACL",
			Interfaces: &vpp_acl.ACL_Interfaces{
				Ingress: []string{dataRequest.Update.VppConfig.Interfaces[0].Name},
			},
			Rules: []*vpp_acl.ACL_Rule{
				//Rule NSMmgmtInterfaceACL permit VXLAN dst
				{
					Action: vpp_acl.ACL_Rule_PERMIT,
					IpRule: &vpp_acl.ACL_Rule_IpRule{
						Ip: &vpp_acl.ACL_Rule_IpRule_Ip{
							DestinationNetwork: v.common.EgressInterface.SrcIPNet().IP.String() + "/32",
							SourceNetwork:      "0.0.0.0/0",
						},
						Udp: &vpp_acl.ACL_Rule_IpRule_Udp{
							DestinationPortRange: &vpp_acl.ACL_Rule_IpRule_PortRange{
								LowerPort: 4789,
								UpperPort: 4789,
							},
							SourcePortRange: &vpp_acl.ACL_Rule_IpRule_PortRange{
								LowerPort: 0,
								UpperPort: 65535,
							},
						},
					},
				},
			},
		},
	}
	logrus.Infof("Setting up Mgmt Interface %v", dataRequest)
	_, err = client.Update(context.Background(), dataRequest)
	if err != nil {
		logrus.Errorf("Error Setting up Mgmt Interface: %s", err)
		return err
	}
	return nil
}

func (v *VPPAgent) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	logrus.Infof("vppagent.DisconnectRequest called with %#v", crossConnect)
	xcon, err := v.connectOrDisconnect(ctx, crossConnect, false)
	if err != nil {
		logrus.Warn(err)
	}
	v.common.Monitor.Delete(ctx, xcon)
	return &empty.Empty{}, err
}

// Init makes setup for the VPPAgent
func (v *VPPAgent) Init(common *common.DataplaneConfig) error {
	v.common = common

	closer := jaeger.InitJaeger(v.common.Name)
	defer func() { _ = closer.Close() }()

	err := v.configureVPPAgent()
	if err != nil {
		logrus.Errorf("Error configuring the VPP Agent: %s", err)
		return err
	}
	return nil
}

func (v *VPPAgent) setupMetricsCollector() {
	if !v.common.MetricsEnabled {
		return
	}
	v.metricsCollector = NewMetricsCollector(v.common.MetricsPeriod)
	v.metricsCollector.CollectAsync(v.common.Monitor, v.vppAgentEndpoint)
}

func (v *VPPAgent) configureVPPAgent() error {
	var ok bool

	v.vppAgentEndpoint, ok = os.LookupEnv(VPPEndpointKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", VPPEndpointKey, VPPEndpointDefault)
		v.vppAgentEndpoint = VPPEndpointDefault
	}
	logrus.Infof("vppAgentEndpoint: %s", v.vppAgentEndpoint)
	if err := nsmonitor.CreateMonitorNetNsInodeServer(v.common.Monitor, v.vppAgentEndpoint); err != nil {
		return err
	}
	v.directMemifConnector = memif.NewDirectMemifConnector(v.common.NSMBaseDir)
	v.common.MechanismsUpdateChannel = make(chan *common.Mechanisms, 1)
	v.common.Mechanisms = &common.Mechanisms{
		LocalMechanisms: []*local.Mechanism{
			{
				Type: local.MechanismType_MEM_INTERFACE,
			},
			{
				Type: local.MechanismType_KERNEL_INTERFACE,
			},
		},
		RemoteMechanisms: []*remote.Mechanism{
			{
				Type: remote.MechanismType_VXLAN,
				Parameters: map[string]string{
					remote.VXLANSrcIP: v.common.EgressInterface.SrcIPNet().IP.String(),
				},
			},
		},
	}
	err := v.reset()
	if err != nil {
		logrus.Errorf("Error resetting the VPP Agent: %s", err)
		return err
	}
	err = v.programMgmtInterface()
	if err != nil {
		logrus.Errorf("Error setting up the management interface for VPP Agent: %s", err)
		return err
	}
	v.setupMetricsCollector()
	return nil
}
