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
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
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

	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	"github.com/networkservicemesh/networkservicemesh/forwarder/pkg/common"
	sdk "github.com/networkservicemesh/networkservicemesh/forwarder/sdk/vppagent"
	"github.com/networkservicemesh/networkservicemesh/forwarder/vppagent/pkg/vppagent/kvschedclient"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

// VPPAgent related constants
const (
	VPPEndpointKey      = "VPPAGENT_ENDPOINT"
	VPPEndpointDefault  = "localhost:9111"
	ManagementInterface = "mgmt"
)

type VPPAgent struct {
	metricsCollector *MetricsCollector
	common           *common.ForwarderConfig
	downstreamResync func()
}

func CreateVPPAgent() *VPPAgent {
	return &VPPAgent{}
}

//CreateForwarderServer creates ForwarderServer handler
func (v *VPPAgent) CreateForwarderServer(config *common.ForwarderConfig) forwarder.ForwarderServer {
	return sdk.ChainOf(
		sdk.ConnectionValidator(),
		sdk.UseMonitor(config.Monitor),
		sdk.DirectMemifInterfaces(config.NSMBaseDir),
		sdk.Connect(v.endpoint()),
		sdk.KernelInterfaces(config.NSMBaseDir),
		sdk.ClearMechanisms(config.NSMBaseDir),
		sdk.Commit(v.downstreamResync),
		sdk.EthernetContextSetter())
}

// MonitorMechanisms sends mechanism updates
func (v *VPPAgent) MonitorMechanisms(empty *empty.Empty, updateSrv forwarder.MechanismsMonitor_MonitorMechanismsServer) error {
	span := spanhelper.FromContext(context.Background(), "MonitorMecnahisms")
	defer span.Finish()
	span.Logger().Infof("MonitorMechanisms was called")
	initialUpdate := &forwarder.MechanismUpdate{
		RemoteMechanisms: v.common.Mechanisms.RemoteMechanisms,
		LocalMechanisms:  v.common.Mechanisms.LocalMechanisms,
	}
	span.Logger().Infof("Sending MonitorMechanisms update: %v", initialUpdate)
	if err := updateSrv.Send(initialUpdate); err != nil {
		span.Logger().Errorf("vpp-agent forwarder server: Detected error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
		return nil
	}
	span.Finish()
	for {
		select {
		// Waiting for any updates which might occur during a life of forwarder module and communicating
		// them back to NSM.
		case update := <-v.common.MechanismsUpdateChannel:
			updateSpan := spanhelper.FromContext(span.Context(), "Sending update")
			v.common.Mechanisms = update
			updateSpan.Logger().Infof("Sending MonitorMechanisms update")
			updateSpan.LogObject("update", update)
			if err := updateSrv.Send(&forwarder.MechanismUpdate{
				RemoteMechanisms: update.RemoteMechanisms,
				LocalMechanisms:  update.LocalMechanisms,
			}); err != nil {
				updateSpan.Finish()
				updateSpan.Logger().Errorf("vpp forwarder server: Detected error %s, grpc code: %+v on grpc channel", err.Error(), status.Convert(err).Code())
				return nil
			}
			updateSpan.Finish()
		}
	}
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
	err := tools.WaitForPortAvailable(ctx, "tcp", v.endpoint(), 100*time.Millisecond)
	if err != nil {
		logrus.Errorf("reset: An error during wait for port available: %v", err.Error())
	}
	conn, err := tools.DialTCPInsecure(v.endpoint())
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
	err := tools.WaitForPortAvailable(ctx, "tcp", v.endpoint(), 100*time.Millisecond)
	if err != nil {
		logrus.Errorf("programMgmtInterface: An error during wait for port available: %v", err.Error())
	}
	conn, err := tools.DialTCPInsecure(v.endpoint())
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
	client := configurator.NewConfiguratorClient(conn)

	vppArpEntries := []*vpp.ARPEntry{}
	vppArpEntriesMap := make(map[string]bool)
	for _, arpEntry := range v.common.EgressInterface.ArpEntries() {
		if _, ok := vppArpEntriesMap[arpEntry.IPAddress]; ok {
			continue
		}
		vppArpEntriesMap[arpEntry.IPAddress] = true
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

// Init makes setup for the VPPAgent
func (v *VPPAgent) Init(common *common.ForwarderConfig) error {
	v.common = common
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
	v.metricsCollector.CollectAsync(v.common.Monitor, v.endpoint())
}

func (v *VPPAgent) endpoint() string {
	return utils.EnvVar(VPPEndpointKey).GetStringOrDefault(VPPEndpointDefault)
}

func (v *VPPAgent) configureVPPAgent() error {
	logrus.Infof("vppAgentEndpoint: %s", v.endpoint())
	var kvSchedulerClient *kvschedclient.KVSchedulerClient
	var err error

	if kvSchedulerClient, err = kvschedclient.NewKVSchedulerClient(v.endpoint()); err != nil {
		return err
	}

	v.downstreamResync = kvSchedulerClient.DownstreamResync
	common.CreateNSMonitor(v.common.Monitor, kvSchedulerClient.DownstreamResync)

	v.common.MechanismsUpdateChannel = make(chan *common.Mechanisms, 1)
	v.common.Mechanisms = &common.Mechanisms{
		LocalMechanisms: []*connection.Mechanism{
			{
				Type: memif.MECHANISM,
			},
			{
				Type: kernel.MECHANISM,
			},
		},
		RemoteMechanisms: []*connection.Mechanism{
			{
				Type: vxlan.MECHANISM,
				Parameters: map[string]string{
					vxlan.SrcIP: v.common.EgressInterface.SrcIPNet().IP.String(),
				},
			},
		},
	}
	err = v.reset()
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
