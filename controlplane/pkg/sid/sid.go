// Copyright (c) 2019 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
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

// Package sid - generate unique sid for connection (fd25::<connectionId>:<nextIndex>)
package sid

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

// SIDPrefix - subnet for generated sid's
const SIDPrefix = "fd25::"

// Allocator - generating unique SID for connection
type Allocator interface {
	SID(requestID string) string
	Restore(hardwareAddr, sid string)
}

type sidAllocator struct {
	lastSID map[string]uint32
	sync.Mutex
}

// NewSIDAllocator - creates sid allocator
func NewSIDAllocator() Allocator {
	return &sidAllocator{
		lastSID: make(map[string]uint32),
	}
}

// SID - Allocate a new SID for SRv6 Policy
func (a *sidAllocator) SID(requestID string) string {
	lastSID := a.lastSID[requestID] + 1
	if lastSID < 2 {
		lastSID = 2
	}

	a.lastSID[requestID] = lastSID
	return fmt.Sprintf("%s:%x", transformRequestID(requestID), lastSID)
}

// Restore value of last SID based on connections we have at the moment
func (a *sidAllocator) Restore(requestID, sid string) {
	parsedSID := net.ParseIP(sid)
	intSID := binary.BigEndian.Uint16(parsedSID[len(parsedSID)-2:])
	a.lastSID[requestID] = uint32(intSID)
}

func transformRequestID(requestID string) string {
	sid := requestID
	for i := 0; i < len(requestID)/4; i++ {
		idx := len(requestID) - (i+1)*4
		sid = fmt.Sprintf("%s:%s", sid[:idx], sid[idx:])
	}

	logrus.Printf("Generated IP: %v %v", fmt.Sprintf("%s%s", SIDPrefix, sid), net.ParseIP(fmt.Sprintf("%s%s", SIDPrefix, sid)).String())

	return net.ParseIP(fmt.Sprintf("%s%s", SIDPrefix, sid)).String()
}
