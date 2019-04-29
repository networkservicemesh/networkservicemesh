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

package vni

import (
	"net"
	"sync"
)

type VniAllocator interface {
	Vni(local_ip string, remote_ip string) uint32
	Restore(local_ip string, remote_ip string, vniId uint32)
}

type vniAllocator struct {
	lastVni map[string]uint32
	sync.Mutex
}

func NewVniAllocator() VniAllocator {
	return &vniAllocator{
		lastVni: make(map[string]uint32),
	}
}

// Vni - Allocate a new VNI, odd if local_ip < remote_ip, even otherwise
func (a *vniAllocator) Vni(local_ip string, remote_ip string) uint32 {
	a.Lock()
	defer a.Unlock()
	lip := net.ParseIP(local_ip)
	rip := net.ParseIP(remote_ip)
	lastVni := a.lastVni[remote_ip]
	if lastVni == 0 {
		if compareIps(lip, rip) < 0 {
			lastVni = 1
		}
	}
	a.lastVni[remote_ip] = lastVni + 2
	return a.lastVni[remote_ip]
}

// Restore value of last Vni based on connections we have at the moment.
func (a *vniAllocator) Restore(local_ip string, remote_ip string, vniId uint32) {
	a.lastVni[remote_ip] = vniId
}


func compareIps(ip1 net.IP, ip2 net.IP) int {
	for index, value := range ip1 {
		if value < ip2[index] {
			return -1
		}
		if value > ip2[index] {
			return 1
		}
	}
	return 0
}
