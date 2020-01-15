package mechanism_kernel_vethpair

import (
	"context"

	linux_interfaces "github.com/ligato/vpp-agent/api/models/linux/interfaces"
	linux_namespace "github.com/ligato/vpp-agent/api/models/linux/namespace"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
)

const (
	ForwarderAllowVHost = "FORWARDER_ALLOW_VHOST" // To disallow VHOST please pass "false" into this env variable.
)

func appendInterfaceConfig(ctx context.Context, conn *connection.Connection, name string) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		conf := vppagent.Config(ctx)
		filepath, err := mechanism.NetNsFileName()
		if err != nil {
			return err
		}
		logrus.Info("Did Not Find /dev/vhost-net - using veth pairs")
		conf.GetLinuxConfig().Interfaces = append(conf.GetLinuxConfig().Interfaces, &linux_interfaces.Interface{
			Name:       name + "-veth",
			Type:       linux_interfaces.Interface_VETH,
			Enabled:    true,
			HostIfName: name + "-veth",
			Link: &linux_interfaces.Interface_Veth{
				Veth: &linux_interfaces.VethLink{
					PeerIfName: name,
				},
			},
		})
		conf.GetLinuxConfig().Interfaces = append(conf.GetLinuxConfig().Interfaces, &linux_interfaces.Interface{
			Name:       name,
			Type:       linux_interfaces.Interface_VETH,
			Enabled:    true,
			HostIfName: mechanism.GetParameters()[common.InterfaceNameKey],
			Namespace: &linux_namespace.NetNamespace{
				Type:      linux_namespace.NetNamespace_FD,
				Reference: filepath,
			},
			Link: &linux_interfaces.Interface_Veth{
				Veth: &linux_interfaces.VethLink{
					PeerIfName: name + "-veth",
				},
			},
		})
		conf.GetVppConfig().Interfaces = append(conf.GetVppConfig().Interfaces, &vpp_interfaces.Interface{
			Name:    name,
			Type:    vpp_interfaces.Interface_AF_PACKET,
			Enabled: true,
			Link: &vpp_interfaces.Interface_Afpacket{
				Afpacket: &vpp_interfaces.AfpacketLink{
					HostIfName: name + "-veth",
				},
			},
		})
	}
	return nil
}
