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
	"fmt"
	"strconv"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"

	"github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/ligato/networkservicemesh/utils/fs"
	"github.com/sirupsen/logrus"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	linux_interfaces "github.com/ligato/vpp-agent/plugins/linux/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
)

const (
	// IPv4Key defines the name of the key ipv4 address in parameters map (optional)
	IPv4Key = "ipv4"
	// IPv4PrefixLengthKey defines the name of the key ipv4 prefix length in parameters map (optional)
	IPv4PrefixLengthKey = "ipv4prefixlength"
	LinuxIfMaxLength    = 15 // The actual value is 15, but best be safe
)

type KernelInterfaceConverter struct {
	*dataplane.Connection
	Side SrcDst
}

func NewKernelInterfaceConverter(c *dataplane.Connection, s SrcDst) Converter {
	rv := &KernelInterfaceConverter{
		Connection: c,
		Side:       s,
	}
	return rv
}

func (c *KernelInterfaceConverter) Name() string {
	return "KernelInterfaceConverter"
}

func (c *KernelInterfaceConverter) Validate() error {
	lm := LocalMechanism(c.Connection, c.Side)
	if lm == nil || lm.Type != common.LocalMechanismType_KERNEL_INTERFACE {
		return fmt.Errorf("Mechanism %#v is not of type KERNEL_INTERFACE", lm)
	}
	if _, ok := lm.Parameters[nsmd.LocalMechanismParameterNetNsInodeKey]; !ok {
		return fmt.Errorf("Missing Required LocalMechanism.Parameter[%s] for network namespace", nsmd.LocalMechanismParameterNetNsInodeKey)
	}
	iface, ok := lm.Parameters[nsmd.LocalMechanismParameterInterfaceNameKey]
	if ok && len(iface) > LinuxIfMaxLength {

	}
	// TODO validated namespace, and IPv4 keys here

	return nil
}

func (c *KernelInterfaceConverter) FullySpecify() error {
	err := c.Validate()
	lm := LocalMechanism(c.Connection, c.Side)
	if err != nil {
		return err
	}
	_, ok := lm.Parameters[nsmd.LocalMechanismParameterInterfaceNameKey]
	if !ok {
		// TODO -  this is a terrible terrible way to name interfaces
		//         ideally, we'd name them nsm-#, but this requires
		//         work to figure out what interfaces we already have
		//         in the namespace
		lm.Parameters[nsmd.LocalMechanismParameterInterfaceNameKey] = c.Side.String() + "-" + c.ConnectionId
	}
	return nil
}

func (c *KernelInterfaceConverter) ToDataRequest(rv *rpc.DataRequest) (*rpc.DataRequest, error) {
	err := c.FullySpecify()
	if rv == nil {
		rv = &rpc.DataRequest{}
	}
	lm := LocalMechanism(c.Connection, c.Side)
	if err != nil {
		return nil, err
	}
	name := c.Side.String() + "-" + c.ConnectionId
	inode, err := strconv.ParseUint(lm.Parameters[nsmd.LocalMechanismParameterNetNsInodeKey], 10, 64)
	if err != nil {
		logrus.Errorf("%s is not an inode number", lm.Parameters[nsmd.LocalMechanismParameterNetNsInodeKey])
		return nil, err
	}
	filepath, err := fs.FindFileInProc(inode, "/ns/net")
	if err != nil {
		logrus.Errorf("No file found in /proc/*/ns/net with inode %d", inode)
		return nil, err
	}
	iface := lm.Parameters[nsmd.LocalMechanismParameterInterfaceNameKey]
	tmpIface := TempIfName()
	logrus.Infof("tmpIface: %s len(tmpIface) %d\n", tmpIface, len(tmpIface))

	description := lm.Parameters[nsmd.LocalMechanismParameterInterfaceDescriptionKey]

	var ipAddresses []string
	if c.Side == SRC && c.ConnectionContext != nil && c.ConnectionContext.ConnectionContext != nil && c.ConnectionContext.ConnectionContext[networkservice.ConnectionContextSrcIPKey] != "" {
		// TODO validate IP address
		ipAddresses = []string{c.ConnectionContext.ConnectionContext[networkservice.ConnectionContextSrcIPKey]}
	}
	if c.Side == DST && c.ConnectionContext != nil && c.ConnectionContext.ConnectionContext != nil && c.ConnectionContext.ConnectionContext[networkservice.ConnectionContextDstIPKey] != "" {
		// TODO validate IP address
		ipAddresses = []string{c.ConnectionContext.ConnectionContext[networkservice.ConnectionContextDstIPKey]}
	}

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
		Name:    name,
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
		Name:        name,
		Type:        linux_interfaces.LinuxInterfaces_AUTO_TAP,
		Enabled:     true,
		Description: description,
		IpAddresses: ipAddresses,
		HostIfName:  iface,
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
