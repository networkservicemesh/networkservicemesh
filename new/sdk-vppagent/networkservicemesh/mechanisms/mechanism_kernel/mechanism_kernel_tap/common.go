package mechanism_kernel_tap

import (
	"context"

	"github.com/ligato/vpp-agent/api/models/linux"
	linux_interfaces "github.com/ligato/vpp-agent/api/models/linux/interfaces"
	linux_namespace "github.com/ligato/vpp-agent/api/models/linux/namespace"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/trace"
)

const (
	ForwarderAllowVHost = "FORWARDER_ALLOW_VHOST" // To disallow VHOST please pass "false" into this env variable.
)

func appendInterfaceConfig(ctx context.Context, conn *connection.Connection, name string) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		conf := vppagent.Config(ctx)
		// We append an Interfaces.  Interfaces creates the vpp side of an interface.
		//   In this case, a Tapv2 interface that has one side in vpp, and the other
		//   as a Linux kernel interface
		conf.GetVppConfig().Interfaces = append(conf.GetVppConfig().Interfaces, &vpp_interfaces.Interface{
			Name:    name,
			Type:    vpp_interfaces.Interface_TAP,
			Enabled: true,
			Link: &vpp_interfaces.Interface_Tap{
				Tap: &vpp_interfaces.TapLink{
					Version: 2,
				},
			},
		})
		filepath, err := mechanism.NetNsFileName()
		if err != nil {
			return err
		}
		trace.Log(ctx).Info("Found /dev/vhost-net - using tapv2")
		// We apply configuration to LinuxInterfaces
		// Important details:
		//    - LinuxInterfaces.HostIfName - must be no longer than 15 chars (linux limitation)
		conf.GetLinuxConfig().Interfaces = append(conf.GetLinuxConfig().Interfaces, &linux.Interface{
			Name:    name,
			Type:    linux_interfaces.Interface_TAP_TO_VPP,
			Enabled: true,
			// TODO - fix this to have a proper getter in the mechanisms/kernel package and use it here
			HostIfName: mechanism.GetParameters()[common.InterfaceNameKey],
			Namespace: &linux_namespace.NetNamespace{
				Type:      linux_namespace.NetNamespace_FD,
				Reference: filepath,
			},
			Link: &linux_interfaces.Interface_Tap{
				Tap: &linux_interfaces.TapLink{
					VppTapIfName: name,
				},
			},
		})
	}
	return nil
}
