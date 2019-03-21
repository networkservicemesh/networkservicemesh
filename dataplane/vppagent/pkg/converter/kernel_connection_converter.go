package converter

import (
	"fmt"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/linux"
	linux_interfaces "github.com/ligato/vpp-agent/api/models/linux/interfaces"
	"github.com/ligato/vpp-agent/api/models/linux/l3"
	"github.com/ligato/vpp-agent/api/models/linux/namespace"
	"github.com/ligato/vpp-agent/api/models/vpp"
	"github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/sirupsen/logrus"
	"os"
)

const DataplaneAllowVHost = "DATAPLANE_ALLOW_VHOST" // To disallow VHOST please pass "false" into this env variable.

type KernelConnectionConverter struct {
	*connection.Connection
	conversionParameters *ConnectionConversionParameters
}

func NewKernelConnectionConverter(c *connection.Connection, conversionParameters *ConnectionConversionParameters) *KernelConnectionConverter {
	return &KernelConnectionConverter{
		Connection:           c,
		conversionParameters: conversionParameters,
	}
}

func (c *KernelConnectionConverter) ToDataRequest(rv *configurator.Config, connect bool) (*configurator.Config, error) {
	if c == nil {
		return rv, fmt.Errorf("LocalConnectionConverter cannot be nil")
	}
	if err := c.IsComplete(); err != nil {
		return rv, err
	}
	if c.GetMechanism().GetType() != connection.MechanismType_KERNEL_INTERFACE {
		return rv, fmt.Errorf("KernelConnectionConverter cannot be used on Connection.Mechanism.Type %s", c.GetMechanism().GetType())
	}
	if rv == nil {
		rv = &configurator.Config{
			LinuxConfig: &linux.ConfigData{},
		}
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}
	if rv.LinuxConfig == nil {
		rv.LinuxConfig = &linux.ConfigData{}
	}

	m := c.GetMechanism()
	filepath, err := m.NetNsFileName()
	if err != nil && connect {
		return nil, err
	}
	tmpIface := TempIfName()

	var ipAddresses []string
	if c.conversionParameters.Side == DESTINATION {
		ipAddresses = []string{c.Connection.GetContext().DstIpAddr}
	}
	if c.conversionParameters.Side == SOURCE {
		ipAddresses = []string{c.Connection.GetContext().SrcIpAddr}
	}

	logrus.Infof("m.GetParameters()[%s]: %s", connection.InterfaceNameKey, m.GetParameters()[connection.InterfaceNameKey])

	// If we have access to /dev/vhost-net, we can use tapv2.  Otherwise fall back to
	// veth pairs
	if useVHostNet() {
		// We append an Interfaces.  Interfaces creates the vpp side of an interface.
		//   In this case, a Tapv2 interface that has one side in vpp, and the other
		//   as a Linux kernel interface
		// Important details:
		//       Interfaces.HostIfName - This is the linux interface name given
		//          to the Linux side of the TAP.  If you wish to apply additional
		//          config like an Ip address, you should make this a random
		//          tmpIface name, and it *must* match the LinuxIntefaces.Tap.TempIfName
		//       Interfaces.Tap.Namespace - do not set this, due to a bug in vppagent
		//          LinuxInterfaces can only be applied if vppagent finds the
		//          interface in vppagent's netns.  So leave it there in the Interfaces
		//          The interface name may be no longer than 15 chars (Linux limitation)
		rv.VppConfig.Interfaces = append(rv.VppConfig.Interfaces, &vpp_interfaces.Interface{
			Name:    c.conversionParameters.Name,
			Type:    vpp_interfaces.Interface_TAP,
			Enabled: true,
			Link: &vpp_interfaces.Interface_Tap{
				Tap: &vpp_interfaces.TapLink{
					Version: 2,
					HostIfName:tmpIface,
				},
			},
		})
		logrus.Info("Found /dev/vhost-net - using tapv2")
		// We apply configuration to LinuxInterfaces
		// Important details:
		//    - If you have created a TAP, LinuxInterfaces.Tap.TempIfName must match
		//      Interfaces.Tap.HostIfName from above
		//    - LinuxInterfaces.HostIfName - must be no longer than 15 chars (linux limitation)
		rv.LinuxConfig.Interfaces = append(rv.LinuxConfig.Interfaces, &linux.Interface{
			Name:        c.conversionParameters.Name,
			Type:        linux_interfaces.Interface_TAP_TO_VPP,
			Enabled:     true,
			//Description: m.GetParameters()[connection.InterfaceDescriptionKey],
			IpAddresses: ipAddresses,
			HostIfName:  m.GetParameters()[connection.InterfaceNameKey],
			Namespace: &linux_namespace.NetNamespace {
				Type:     linux_namespace.NetNamespace_FD,
				Reference: filepath,
			},
			Link: &linux_interfaces.Interface_Tap {
				Tap: &linux_interfaces.TapLink{
					VppTapIfName: tmpIface,
				},
			},
		})
	} else {
		logrus.Info("Did Not Find /dev/vhost-net - using veth pairs")
		rv.LinuxConfig.Interfaces = append(rv.LinuxConfig.Interfaces, &linux_interfaces.Interface{
			Name:        tmpIface,
			Type:        linux_interfaces.Interface_VETH,
			Enabled:     true,
			//Description: m.GetParameters()[connection.InterfaceDescriptionKey],
			IpAddresses: ipAddresses,
			HostIfName:  tmpIface,
			Link: &linux_interfaces.Interface_Veth{
				Veth: &linux_interfaces.VethLink{
					PeerIfName: c.conversionParameters.Name,
				},
			},
		})
		rv.LinuxConfig.Interfaces = append(rv.LinuxConfig.Interfaces, &linux_interfaces.Interface{
			Name:        c.conversionParameters.Name,
			Type:        linux_interfaces.Interface_VETH,
			Enabled:     true,
			//Description: m.GetParameters()[connection.InterfaceDescriptionKey],
			IpAddresses: ipAddresses,
			HostIfName:  m.GetParameters()[connection.InterfaceNameKey],
			Namespace: &linux_namespace.NetNamespace {
				Type:     linux_namespace.NetNamespace_FD,
				Reference: filepath,
			},
			Link: &linux_interfaces.Interface_Veth{
				Veth: &linux_interfaces.VethLink{
					PeerIfName:	tmpIface,
				},
			},
		})
		rv.VppConfig.Interfaces = append(rv.VppConfig.Interfaces, &vpp_interfaces.Interface{
			Name:    c.conversionParameters.Name,
			Type:    vpp_interfaces.Interface_AF_PACKET,
			Enabled: true,
			Link: &vpp_interfaces.Interface_Afpacket {
				Afpacket: &vpp_interfaces.AfpacketLink{
					HostIfName: tmpIface,
				},
			},
		})

	}

	// Process static routes
	if c.conversionParameters.Side == SOURCE {
		for _, route := range c.Connection.GetContext().GetRoutes() {
			rv.LinuxConfig.Routes = append(rv.LinuxConfig.Routes, &linux.Route{
				DstNetwork:   route.Prefix,
				//Description: "Route to " + route.Prefix,
				OutgoingInterface:   c.conversionParameters.Name,
				Scope: linux_l3.Route_HOST,
				GwAddr: extractCleanIPAddress(c.Connection.GetContext().DstIpAddr),
			})
		}
	}

	// Process IP Neighbor entries
	if c.conversionParameters.Side == SOURCE {
		for _, neightbour := range c.Connection.GetContext().GetIpNeighbors() {
			rv.LinuxConfig.ArpEntries = append(rv.LinuxConfig.ArpEntries, &linux.ARPEntry{
				//Name:      fmt.Sprintf("%s_arp_%d", c.conversionParameters.Name, idx),
				IpAddress:    neightbour.Ip,
				Interface: c.conversionParameters.Name,
				HwAddress: neightbour.HardwareAddress,
			})
		}
	}

	return rv, nil
}

func useVHostNet() bool {
	vhostAllowed := os.Getenv(DataplaneAllowVHost)
	if "false" == vhostAllowed {
		return false
	}
	if _, err := os.Stat("/dev/vhost-net"); err == nil {
		return true
	}
	return false
}
