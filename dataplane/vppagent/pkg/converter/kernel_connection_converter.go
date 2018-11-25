package converter

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	linux_interfaces "github.com/ligato/vpp-agent/plugins/linux/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
)

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

func (c *KernelConnectionConverter) ToDataRequest(rv *rpc.DataRequest) (*rpc.DataRequest, error) {
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
	if err != nil {
		return nil, err
	}
	tmpIface := TempIfName()

	var ipAddresses []string
	if c.conversionParameters.Side == DESTINATION {
		ipAddresses = []string{c.Connection.GetContext()[connectioncontext.DstIpKey]}
	}
	if c.conversionParameters.Side == SOURCE {
		ipAddresses = []string{c.Connection.GetContext()[connectioncontext.SrcIpKey]}
	}

	logrus.Infof("m.GetParameters()[%s]: %s", connection.InterfaceNameKey, m.GetParameters()[connection.InterfaceNameKey])

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
		IpAddresses: ipAddresses, // TODO - this is wrong... need to use Dst or Src key selectively
		HostIfName:  m.GetParameters()[connection.InterfaceNameKey],
		Namespace: &linux_interfaces.LinuxInterfaces_Interface_Namespace{
			Type:     linux_interfaces.LinuxInterfaces_Interface_Namespace_FILE_REF_NS,
			Filepath: filepath,
		},
		Tap: &linux_interfaces.LinuxInterfaces_Interface_Tap{
			TempIfName: tmpIface,
		},
	})
	return rv, nil
}
