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
	govppapi "git.fd.io/govpp.git/api"
)

type interfaceUpDown struct {
	intf   *vppInterface
	upDown uint8
}

func (op *interfaceUpDown) apply(apiCh govppapi.Channel) error {
	// TODO
	return nil
}

func (op *interfaceUpDown) rollback() operation {
	var upDown uint8
	if op.upDown == 1 {
		upDown = 0
	} else {
		upDown = 1
	}
	return &interfaceUpDown{
		intf:   op.intf,
		upDown: upDown,
	}
}

type interfaceXconnect struct {
	rx     *vppInterface
	tx     *vppInterface
	enable uint8
}

func (op *interfaceXconnect) apply(apiCh govppapi.Channel) error {
	// TODO
	// xconnectReq := l2.SwInterfaceSetL2Xconnect{
	// 	RxSwIfIndex: op.rx.id,
	// 	TxSwIfIndex: op.tx.id,
	// 	Enable:      op.enable,
	// }
	// xconnectRpl := l2.SwInterfaceSetL2XconnectReply{}
	// if err := apiCh.SendRequest(&xconnectReq).ReceiveReply(&xconnectRpl); err != nil {
	// 	return err
	// }
	return nil
}

func (op *interfaceXconnect) rollback() operation {
	// var enable uint8
	// if op.enable == 1 {
	// 	enable = 0
	// } else {
	// 	enable = 1
	// }
	// TODO
	// return &interfaceXconnect{
	// 	rx:     op.rx,
	// 	tx:     op.tx,
	// 	enable: enable,
	// }
	return nil
}
