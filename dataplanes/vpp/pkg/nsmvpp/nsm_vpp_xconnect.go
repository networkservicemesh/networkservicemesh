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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/bin_api/interfaces"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/bin_api/l2"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/sirupsen/logrus"
)

// DreateLocalConnect sanity checks parameters passed in the LocalMechanisms and call nsmvpp.CreateLocalConnect
func CreateLocalConnect(apiCh govppapi.Channel, src, dst *common.LocalMechanism) (string, error) {
	srcMechanism := mechanisms[src.Type]
	dstMechanism := mechanisms[dst.Type]

	// validate both src and dst parameters beforehands so we unlikely in situation when
	// we created one interface and failed to create another...
	// TODO anyway, this situaition is possible -- need to implement rollback mechanism
	if err := srcMechanism.validate(src.Parameters); err != nil {
		return "", err
	}
	if err := dstMechanism.validate(dst.Parameters); err != nil {
		return "", err
	}

	srcIntf, err := srcMechanism.createInterface(apiCh, src.Parameters)
	if err != nil {
		logrus.Errorf("failure during creation of source interface, error: %+v", err)
		return "", fmt.Errorf("Error in reply: %+v", err)

	}
	dstIntf, err := dstMechanism.createInterface(apiCh, dst.Parameters)
	if err != nil {
		logrus.Errorf("failure during creation of destination interface, error: %+v", err)
		return "", fmt.Errorf("Error in reply: %+v", err)

	}

	// Cross connecting two taps
	if err := buildCrossConnect(apiCh, srcIntf, dstIntf); err != nil {
		logrus.Errorf("failure during creation of a cross connect, with error: %+v", err)
		return "", fmt.Errorf("Error in reply: %+v", err)
	}

	// Bring both tap interfaces up
	if err := bringInterfaceUp(apiCh, srcIntf); err != nil {
		logrus.Errorf("failure to bring src interface up, with error: %+v", err)
		return "", fmt.Errorf("Error in reply: %+v", err)
	}

	if err := bringInterfaceUp(apiCh, dstIntf); err != nil {
		logrus.Errorf("failure to bring dst interface up, with error: %+v", err)
		return "", fmt.Errorf("Error in reply: %+v", err)
	}

	logrus.Infof("Cross connect for interfaces ID %d and %d successful.", srcIntf, dstIntf)

	// Do not like this, this is about to encode connection information
	// into connection id, which is required to bring connection down
	// later we need to track this information on vpp side
	return fmt.Sprintf("%d-%d-%d-%d", src.Type, srcIntf, dst.Type, dstIntf), nil
}

// DeleteLocalConnect
func DeleteLocalConnect(apiCh govppapi.Channel, connID string) error {
	info := strings.Split(connID, "-")

	srcKind, _ := strconv.Atoi(info[0])
	srcI, _ := strconv.Atoi(info[1])
	dstKind, _ := strconv.Atoi(info[2])
	dstI, _ := strconv.Atoi(info[3])

	srcIntf := uint32(srcI)
	dstIntf := uint32(dstI)

	// Bring both interfaces down
	if err := bringInterfaceDown(apiCh, dstIntf); err != nil {
		logrus.Errorf("failure to bring tap interface Up, with error: %+v", err)
		return fmt.Errorf("Error in reply: %+v", err)
	}

	if err := bringInterfaceDown(apiCh, srcIntf); err != nil {
		logrus.Errorf("failure to bring tap interface Up, with error: %+v", err)
		return fmt.Errorf("Error in reply: %+v", err)
	}

	// Remove Cross connect from two taps
	if err := removeCrossConnect(apiCh, srcIntf, dstIntf); err != nil {
		logrus.Errorf("failure during creation of a cross connect, with error: %+v", err)
		return fmt.Errorf("Error in reply: %+v", err)
	}

	// Delete vpp interfaces
	if err := mechanisms[srcKind].deleteInterface(apiCh, srcIntf); err != nil {
		logrus.Errorf("failure during creation of a source tap, with error: %+v", err)
		return fmt.Errorf("Error in reply: %+v", err)
	}
	if err := mechanisms[dstKind].deleteInterface(apiCh, dstIntf); err != nil {
		logrus.Errorf("failure during creation of a source tap, with error: %+v", err)
		return fmt.Errorf("Error in reply: %+v", err)
	}

	return nil
}

// bringInterfaceUp
func bringInterfaceUp(apiCh govppapi.Channel, intf uint32) error {
	if err := apiCh.SendRequest(&interfaces.SwInterfaceSetFlags{
		SwIfIndex:   intf,
		AdminUpDown: 1,
	}).ReceiveReply(&interfaces.SwInterfaceSetFlagsReply{}); err != nil {
		return err
	}
	return nil
}

// build CrossConnect creates a 2 one way xconnect circuits between two tap interfaces
func buildCrossConnect(apiCh govppapi.Channel, intf1, intf2 uint32) error {
	xconnectReq := l2.SwInterfaceSetL2Xconnect{
		RxSwIfIndex: intf1,
		TxSwIfIndex: intf2,
		Enable:      1,
	}
	xconnectRpl := l2.SwInterfaceSetL2XconnectReply{}
	if err := apiCh.SendRequest(&xconnectReq).ReceiveReply(&xconnectRpl); err != nil {
		return err
	}
	xconnectReq = l2.SwInterfaceSetL2Xconnect{
		RxSwIfIndex: intf2,
		TxSwIfIndex: intf1,
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

// bringCrossConnectDown brings Down both tap interfaces which are a part of CrossConnect
func bringInterfaceDown(apiCh govppapi.Channel, intf uint32) error {
	if err := apiCh.SendRequest(&interfaces.SwInterfaceSetFlags{
		SwIfIndex:   intf,
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
