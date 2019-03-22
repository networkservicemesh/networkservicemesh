package converter

import (
	"fmt"
	linux_interfaces "github.com/ligato/vpp-agent/plugins/linux/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/linux/model/l3"
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/sirupsen/logrus"
	"os"
)

const DataplaneAllowVHost = "DATAPLANE_ALLOW_VHOST"	// To disallow VHOST please pass "false" into this env variable.

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

func (c *KernelConnectionConverter) ToDataRequest(rv *rpc.DataRequest, connect bool) (*rpc.DataRequest, error) {
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
		rv = &rpc.DataRequest{}
	}

	m := c.GetMechanism()
	filepath, err := m.NetNsFileName()
	if err != nil && connect {
		return nil, err
	}
	tmpIface := c.conversionParameters.IfaceNameProvider.GetIfaceName(c.conversionParameters.Name)

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
		rv.Interfaces = append(rv.Interfaces, &interfaces.Interfaces_Interface{
			Name:    c.conversionParameters.Name,
			Type:    interfaces.InterfaceType_TAP_INTERFACE,
			Enabled: true,
			Tap: &interfaces.Interfaces_Interface_Tap{
				Version:    2,
				HostIfName: tmpIface,
			},
		})
		logrus.Info("Found /dev/vhost-net - using tapv2")
		// We apply configuration to LinuxInterfaces
		// Important details:
		//    - If you have created a TAP, LinuxInterfaces.Tap.TempIfName must match
		//      Interfaces.Tap.HostIfName from above
		//    - LinuxInterfaces.HostIfName - must be no longer than 15 chars (linux limitation)
		rv.LinuxInterfaces = append(rv.LinuxInterfaces, &linux_interfaces.LinuxInterfaces_Interface{
			Name:        c.conversionParameters.Name,
			Type:        linux_interfaces.LinuxInterfaces_AUTO_TAP,
			Enabled:     true,
			Description: m.GetParameters()[connection.InterfaceDescriptionKey],
			IpAddresses: ipAddresses,
			HostIfName:  m.GetParameters()[connection.InterfaceNameKey],
			Namespace: &linux_interfaces.LinuxInterfaces_Interface_Namespace{
				Type:     linux_interfaces.LinuxInterfaces_Interface_Namespace_FILE_REF_NS,
				Filepath: filepath,
			},
			Tap: &linux_interfaces.LinuxInterfaces_Interface_Tap{
				TempIfName: tmpIface,
			},
		})
	} else {
		logrus.Info("Did Not Find /dev/vhost-net - using veth pairs")
		rv.LinuxInterfaces = append(rv.LinuxInterfaces, &linux_interfaces.LinuxInterfaces_Interface{
			Name:        tmpIface,
			Type:        linux_interfaces.LinuxInterfaces_VETH,
			Enabled:     true,
			Description: m.GetParameters()[connection.InterfaceDescriptionKey],
			IpAddresses: ipAddresses,
			HostIfName:  tmpIface,
			Veth: &linux_interfaces.LinuxInterfaces_Interface_Veth{
				PeerIfName: c.conversionParameters.Name,
			},
		})
		rv.LinuxInterfaces = append(rv.LinuxInterfaces, &linux_interfaces.LinuxInterfaces_Interface{
			Name:        c.conversionParameters.Name,
			Type:        linux_interfaces.LinuxInterfaces_VETH,
			Enabled:     true,
			Description: m.GetParameters()[connection.InterfaceDescriptionKey],
			IpAddresses: ipAddresses,
			HostIfName:  m.GetParameters()[connection.InterfaceNameKey],
			Namespace: &linux_interfaces.LinuxInterfaces_Interface_Namespace{
				Type:     linux_interfaces.LinuxInterfaces_Interface_Namespace_FILE_REF_NS,
				Filepath: filepath,
			},
			Veth: &linux_interfaces.LinuxInterfaces_Interface_Veth{
				PeerIfName: tmpIface,
			},
		})
		rv.Interfaces = append(rv.Interfaces, &interfaces.Interfaces_Interface{
			Name:    c.conversionParameters.Name,
			Type:    interfaces.InterfaceType_AF_PACKET_INTERFACE,
			Enabled: true,
			Afpacket: &interfaces.Interfaces_Interface_Afpacket{
				HostIfName: tmpIface,
			},
		})

	}

	// Process static routes
	if c.conversionParameters.Side == SOURCE {
		for idx, route := range c.Connection.GetContext().GetRoutes() {
			rv.LinuxRoutes = append(rv.LinuxRoutes, &l3.LinuxStaticRoutes_Route{
				Name:        fmt.Sprintf("%s_route_%d", c.conversionParameters.Name, idx),
				DstIpAddr:   route.Prefix,
				Description: "Route to " + route.Prefix,
				Interface:   c.conversionParameters.Name,
				Namespace: &l3.LinuxStaticRoutes_Route_Namespace{
					Type:     l3.LinuxStaticRoutes_Route_Namespace_FILE_REF_NS,
					Filepath: filepath,
				},
				GwAddr: extractCleanIPAddress(c.Connection.GetContext().DstIpAddr),
			})
		}
	}

	// Process IP Neighbor entries
	if c.conversionParameters.Side == SOURCE {
		for idx, neightbour := range c.Connection.GetContext().GetIpNeighbors() {
			rv.LinuxArpEntries = append(rv.LinuxArpEntries, &l3.LinuxStaticArpEntries_ArpEntry{
				Name:      fmt.Sprintf("%s_arp_%d", c.conversionParameters.Name, idx),
				IpAddr:    neightbour.Ip,
				Interface: c.conversionParameters.Name,
				HwAddress: neightbour.HardwareAddress,
				Namespace: &l3.LinuxStaticArpEntries_ArpEntry_Namespace{
					Type:     l3.LinuxStaticArpEntries_ArpEntry_Namespace_FILE_REF_NS,
					Filepath: filepath,
				},
				State: &l3.LinuxStaticArpEntries_ArpEntry_NudState{
					Type: l3.LinuxStaticArpEntries_ArpEntry_NudState_PERMANENT, // or NOARP, REACHABLE, STALE
				},
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
