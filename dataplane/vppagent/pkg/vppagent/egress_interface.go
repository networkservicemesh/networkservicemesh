// Copyright (c) 2019 Cisco and/or its affiliates.
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
	"net"
)

type EgressInterface struct {
	*net.Interface
	srcNet *net.IPNet
}

func NewEgressInterface(srcIp net.IP) (*EgressInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.Equal(srcIp) {
					return &EgressInterface{
						srcNet:    v,
						Interface: &iface,
					}, nil
				}
			default:
				return nil, fmt.Errorf("Type of addr not net.IPNET")
			}
		}
	}
	return nil, fmt.Errorf("Unable to find interface with IP: %s", srcIp)
}

func (e *EgressInterface) SrcIPNet() *net.IPNet {
	if e == nil {
		return nil
	}
	return e.srcNet
}
