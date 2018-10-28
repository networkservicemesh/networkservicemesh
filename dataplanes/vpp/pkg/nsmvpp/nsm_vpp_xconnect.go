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

package nsmvpp

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/bin_api/interfaces"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/bin_api/l2"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/bin_api/tapv2"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/utils/fs"
	"github.com/sirupsen/logrus"
)

type tapInterface struct {
	id           uint32
	name         string
	namespace    []byte
	ip           []byte
	prefixLength uint8
	tag          []byte
}

type KernelInterface struct{}

var keyList = nsmutils.Keys{
	nsmutils.NSMkeyNamespace: nsmutils.KeyProperties{
		Mandatory: true,
		Validator: nsmutils.Namespace},
	nsmutils.NSMkeyIPv4: nsmutils.KeyProperties{
		Mandatory: false,
		Validator: nsmutils.Ipv4},
	nsmutils.NSMkeyIPv4PrefixLength: nsmutils.KeyProperties{
		Mandatory: false,
		Validator: nsmutils.Ipv4prefixlength},
}

func encode(s []byte) string {
	md5 := md5.Sum(s)
	return hex.EncodeToString(md5[:4])
}

// CreateLocalConnect creates two tap interfaces in corresponding namespaces and then cross connect them
func (m KernelInterface) CreateLocalConnect(apiCh govppapi.Channel, srcParameters, dstParameters map[string]string) (string, error) {
	var err error

	if err := m.Validate(srcParameters); err != nil {
		return "", err
	}
	if err := m.Validate(dstParameters); err != nil {
		return "", err
	}
	// Extract namespaces for source and destination containers
	srcNamespace := srcParameters[nsmutils.NSMkeyNamespace]
	dstNamespace := dstParameters[nsmutils.NSMkeyNamespace]

	if !strings.HasPrefix(srcNamespace, "pid:") {
		// assuming that inode of linux namespace has been passed
		inode, err := strconv.ParseUint(srcNamespace, 10, 64)
		if err != nil {
			logrus.Errorf("can't parse integer: %s", srcNamespace)
		} else {
			srcNamespace, err = fs.FindFileInProc(inode, "/ns/net")
			if err != nil {
				logrus.Errorf("cant' find namespace for inode %d", inode)
				return "", err
			}
		}
	}

	if !strings.HasPrefix(dstNamespace, "pid:") {
		// assuming that inode of linux namespace has been passed
		inode, err := strconv.ParseUint(dstNamespace, 10, 64)
		if err != nil {
			logrus.Errorf("can't parse integer: %s", dstNamespace)
		} else {
			dstNamespace, err = fs.FindFileInProc(inode, "/ns/net")
			if err != nil {
				logrus.Errorf("cant' find namespace for inode %d", inode)
				return "", err
			}
		}
	}

	logrus.Infof("connecting namespaces %s and %s...", srcNamespace, dstNamespace)

	tap1 := &tapInterface{
		namespace: []byte(srcNamespace),
		tag:       []byte("NSM_CLIENT"),
	}
	tap2 := &tapInterface{
		namespace: []byte(dstNamespace),
		tag:       []byte("NSM_CLIENT"),
	}
	tap1.name = "tap-" + encode(tap2.namespace)
	tap2.name = "tap-" + encode(tap1.namespace)

	// Making sure that total interface name length is not exceeding 15 bytes.
	if len(tap1.name) > 15 {
		tap1.name = tap1.name[:15]
		tap2.name = tap2.name[:15]
	}
	logrus.Infof("Resulting tap interface names: %s %s", tap1.name, tap2.name)

	// This block check for ipv4 addresses in Parameters map, if specified, it verifies that both either present or
	// both missing and populate tap struct wit hcorresponding fields.
	srcIPv4, b1 := srcParameters[nsmutils.NSMkeyIPv4]
	dstIPv4, b2 := dstParameters[nsmutils.NSMkeyIPv4]
	if b1 != b2 {
		return "", fmt.Errorf("both containers must either specify or both must not specify ipv4 addresses")
	}
	// since both b1 and b2 are == sufficient to check just b1 is it is true or not
	// in case of true, add requested ipv4 addresses
	if b1 {
		if tap1.ip, err = IPv4ToByteSlice(srcIPv4); err != nil {
			return "", err
		}
		// Safe to ignore converstion error as ValidateParameters has validated already success of conversion.
		l, _ := strconv.Atoi(srcParameters[nsmutils.NSMkeyIPv4PrefixLength])
		tap1.prefixLength = uint8(l)
		if tap2.ip, err = IPv4ToByteSlice(dstIPv4); err != nil {
			return "", err
		}
		l, _ = strconv.Atoi(srcParameters[nsmutils.NSMkeyIPv4PrefixLength])
		tap2.prefixLength = uint8(l)
	}

	// Creating TAP interfaces
	if err := createTapInterface(apiCh, tap1); err != nil {
		logrus.Errorf("failure during creation of a source tap, with error: %+v", err)
		return "", fmt.Errorf("Error in reply: %+v", err)
	}
	if err := createTapInterface(apiCh, tap2); err != nil {
		logrus.Errorf("failure during creation of a source tap, with error: %+v", err)
		return "", fmt.Errorf("Error in reply: %+v", err)
	}

	// Cross connecting two taps
	if err := buildCrossConnect(apiCh, tap1, tap2); err != nil {
		logrus.Errorf("failure during creation of a cross connect, with error: %+v", err)
		return "", fmt.Errorf("Error in reply: %+v", err)
	}
	logrus.Infof("Cross connect for interfaces ID %d and %d was creation was succesful", tap1.id, tap2.id)

	// Bring both tap interfaces up
	if err := bringCrossConnectUp(apiCh, tap1, tap2); err != nil {
		logrus.Errorf("failure to bring tap interface Up, with error: %+v", err)
		return "", fmt.Errorf("Error in reply: %+v", err)
	}

	return fmt.Sprintf("%d-%x-%x", common.LocalMechanismType_KERNEL_INTERFACE, tap1.id, tap2.id), nil
}

func (m KernelInterface) Validate(parameters map[string]string) error {
	// Check presence of both ipv4 address and prefix length
	_, v1 := parameters[nsmutils.NSMkeyIPv4]
	_, v2 := parameters[nsmutils.NSMkeyIPv4PrefixLength]
	if v1 != v2 {
		return fmt.Errorf("both parameter \"ipv4\" and \"ipv4prefixlength\" must either present or missing")
	}

	return nsmutils.ValidateParameters(parameters, keyList)
}

// createTapInterface creates new tap interface in a specified namespace
func createTapInterface(apiCh govppapi.Channel, tap *tapInterface) error {
	var tapReq tapv2.TapCreateV2
	var tapRpl tapv2.TapCreateV2Reply
	tapReq.ID = ^uint32(0)
	tapReq.HostNamespaceSet = 1
	tapReq.HostNamespace = tap.namespace
	tapReq.UseRandomMac = 1
	tapReq.Tag = tap.tag
	tapReq.HostIfName = []byte(tap.name)
	tapReq.HostIfNameSet = 1
	if len(tap.ip) != 0 {
		tapReq.HostIP4Addr = tap.ip
		tapReq.HostIP4PrefixLen = tap.prefixLength
		tapReq.HostIP4AddrSet = 1
	}
	if err := apiCh.SendRequest(&tapReq).ReceiveReply(&tapRpl); err != nil {
		return err
	}
	tap.id = tapRpl.SwIfIndex
	return nil
}

// bringCrossConnectUp brings Up both tap interfaces which are a part of CrossConnect
func bringCrossConnectUp(apiCh govppapi.Channel, tap1, tap2 *tapInterface) error {
	if err := apiCh.SendRequest(&interfaces.SwInterfaceSetFlags{
		SwIfIndex:   tap1.id,
		AdminUpDown: 1,
	}).ReceiveReply(&interfaces.SwInterfaceSetFlagsReply{}); err != nil {
		return err
	}
	if err := apiCh.SendRequest(&interfaces.SwInterfaceSetFlags{
		SwIfIndex:   tap2.id,
		AdminUpDown: 1,
	}).ReceiveReply(&interfaces.SwInterfaceSetFlagsReply{}); err != nil {
		return err
	}
	return nil
}

// build CrossConnect creates a 2 one way xconnect circuits between two tap interfaces
func buildCrossConnect(apiCh govppapi.Channel, tap1, tap2 *tapInterface) error {
	xconnectReq := l2.SwInterfaceSetL2Xconnect{
		RxSwIfIndex: tap1.id,
		TxSwIfIndex: tap2.id,
		Enable:      1,
	}
	xconnectRpl := l2.SwInterfaceSetL2XconnectReply{}
	if err := apiCh.SendRequest(&xconnectReq).ReceiveReply(&xconnectRpl); err != nil {
		return err
	}
	xconnectReq = l2.SwInterfaceSetL2Xconnect{
		RxSwIfIndex: tap2.id,
		TxSwIfIndex: tap1.id,
		Enable:      1,
	}
	xconnectRpl = l2.SwInterfaceSetL2XconnectReply{}
	if err := apiCh.SendRequest(&xconnectReq).ReceiveReply(&xconnectRpl); err != nil {
		return err
	}
	return nil
}

// IPv4ToByteSlice converts an ipv4 address in form '1.2.3.4' to an []byte]
// representation of the address.
func IPv4ToByteSlice(ipv4Address string) ([]byte, error) {
	var ipu []byte

	ipv4Address = strings.Trim(ipv4Address, " ")
	match, _ := regexp.Match(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`,
		[]byte(ipv4Address))
	if !match {
		return nil, fmt.Errorf("invalid IP address %s", ipv4Address)
	}
	parts := strings.Split(ipv4Address, ".")
	for _, p := range parts {
		num, _ := strconv.Atoi(p)
		ipu = append(ipu, byte(num))
	}

	return ipu, nil
}

// DeleteLocalConnect creates two tap interfaces in corresponding namespaces and then cross connect them
func (m KernelInterface) DeleteLocalConnect(apiCh govppapi.Channel, connID string) error {
	t1, _ := strconv.Atoi(strings.Split(connID, "-")[1])
	t2, _ := strconv.Atoi(strings.Split(connID, "-")[2])
	tap1IntfID := uint32(t1)
	tap2IntfID := uint32(t2)

	// Bring both tap interfaces down
	if err := bringCrossConnectDown(apiCh, tap1IntfID, tap2IntfID); err != nil {
		logrus.Errorf("failure to bring tap interface Up, with error: %+v", err)
		return fmt.Errorf("Error in reply: %+v", err)
	}

	// Remove Cross connect from two taps
	if err := removeCrossConnect(apiCh, tap1IntfID, tap2IntfID); err != nil {
		logrus.Errorf("failure during creation of a cross connect, with error: %+v", err)
		return fmt.Errorf("Error in reply: %+v", err)
	}

	// Delete TAP interfaces
	if err := deleteTapInterface(apiCh, tap1IntfID); err != nil {
		logrus.Errorf("failure during creation of a source tap, with error: %+v", err)
		return fmt.Errorf("Error in reply: %+v", err)
	}
	if err := deleteTapInterface(apiCh, tap2IntfID); err != nil {
		logrus.Errorf("failure during creation of a source tap, with error: %+v", err)
		return fmt.Errorf("Error in reply: %+v", err)
	}

	return nil
}

// deleteTapInterface creates new tap interface in a specified namespace
func deleteTapInterface(apiCh govppapi.Channel, tapIntfID uint32) error {
	var tapReq tapv2.TapDeleteV2
	tapReq.SwIfIndex = tapIntfID
	if err := apiCh.SendRequest(&tapReq).ReceiveReply(&tapv2.TapDeleteV2Reply{}); err != nil {
		return err
	}
	return nil
}

// bringCrossConnectDown brings Down both tap interfaces which are a part of CrossConnect
func bringCrossConnectDown(apiCh govppapi.Channel, tap1IntfID, tap2IntfID uint32) error {
	if err := apiCh.SendRequest(&interfaces.SwInterfaceSetFlags{
		SwIfIndex:   tap1IntfID,
		AdminUpDown: 0,
	}).ReceiveReply(&interfaces.SwInterfaceSetFlagsReply{}); err != nil {
		return err
	}
	if err := apiCh.SendRequest(&interfaces.SwInterfaceSetFlags{
		SwIfIndex:   tap2IntfID,
		AdminUpDown: 0,
	}).ReceiveReply(&interfaces.SwInterfaceSetFlagsReply{}); err != nil {
		return err
	}
	return nil
}

// removeCrossConnect removes  2 one way xconnect circuits between two tap interfaces
func removeCrossConnect(apiCh govppapi.Channel, tap1IntfID, tap2IntfID uint32) error {
	xconnectReq := l2.SwInterfaceSetL2Xconnect{
		RxSwIfIndex: tap1IntfID,
		TxSwIfIndex: tap2IntfID,
		Enable:      0,
	}
	xconnectRpl := l2.SwInterfaceSetL2XconnectReply{}
	if err := apiCh.SendRequest(&xconnectReq).ReceiveReply(&xconnectRpl); err != nil {
		return err
	}
	xconnectReq = l2.SwInterfaceSetL2Xconnect{
		RxSwIfIndex: tap2IntfID,
		TxSwIfIndex: tap1IntfID,
		Enable:      0,
	}
	xconnectRpl = l2.SwInterfaceSetL2XconnectReply{}
	if err := apiCh.SendRequest(&xconnectReq).ReceiveReply(&xconnectRpl); err != nil {
		return err
	}
	return nil
}
