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
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

func LocalMechanism(c *dataplane.CrossConnect, s SrcDst) *connection.Mechanism {
	var rv *connection.Mechanism
	if s == SRC {
		src, ok := c.GetSource().(*dataplane.CrossConnect_LocalSource)
		if ok {
			rv = src.LocalSource.GetMechanism()
		}
	}
	if s == DST {
		dst, ok := c.GetDestination().(*dataplane.CrossConnect_LocalDestination)
		if ok {
			rv = dst.LocalDestination.GetMechanism()
		}
	}
	return rv
}

func TempIfName() string {
	// xids are  12 bytes -
	// 4-byte value representing the seconds since the Unix epoch,
	// 3-byte machine identifier,
	// 2-byte process id, and
	// 3-byte counter, starting with a random value.
	guid := xid.New()

	// We need something randomish but not more than 15 bytes
	// Obviously we only care about the first 4 bytes and the last 3 bytes
	// xid encodes to base32, so each char represents 5 bits
	// 4*8/5 =  6.4 - so if we grab the first 7 chars thats going to include
	// the first four bytes
	// 3*8/5 = 4.8 - so if we grab the last 5 chars that will include the
	// last three bytes

	rv := guid.String()
	rv = rv[:7] + rv[16:]
	logrus.Infof("Generated unique TempIfName: %s len(TempIfName) %d", rv, len(rv))
	return rv
}
