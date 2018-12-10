// Copyright 2018 VMware, Inc.
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

package nscomposer

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
)

const (
	// NetworkServiceName defines Network Service Name the NSE is serving for
	advertiseNseNameEnv     = "ADVERTISE_NSE_NAME"
	advertiseNseLabelsEnv   = "ADVERTISE_NSE_LABELS"
	outgoingNscNameEnv      = "OUTGOING_NSC_NAME"
	outgoingNscLabelsEnv    = "OUTGOING_NSC_LABELS"
	outgoingNscMechanismEnv = "OUTGOING_NSC_MECHANISM"
	ipAddressEnv            = "IP_ADDRESS"
)

// NSConfiguration contains the fill configuration of
type NSConfiguration struct {
	nsmServerSocket      string
	nsmClientSocket      string
	workspace            string
	AdvertiseNseName     string
	OutgoingNscName      string
	AdvertiseNseLabels   string
	OutgoingNscLabels    string
	OutgoingNscMechanism string
	IPAddress            string
}

func (configuration *NSConfiguration) CompleteNSConfiguration() {

	if len(configuration.nsmServerSocket) == 0 {
		configuration.nsmServerSocket = getEnv(nsmd.NsmServerSocketEnv, "nsmServerSocket", true)
	}

	if len(configuration.nsmClientSocket) == 0 {
		configuration.nsmClientSocket = getEnv(nsmd.NsmClientSocketEnv, "nsmClientSocket", true)
	}

	if len(configuration.workspace) == 0 {
		configuration.workspace = getEnv(nsmd.WorkspaceEnv, "workspace", true)
	}

	if len(configuration.AdvertiseNseName) == 0 {
		configuration.AdvertiseNseName = getEnv(advertiseNseNameEnv, "Advertise Network Service Name", false)
	}

	if len(configuration.OutgoingNscName) == 0 {
		configuration.OutgoingNscName = getEnv(outgoingNscNameEnv, "Outgoing Network Service Name", false)
	}

	if len(configuration.AdvertiseNseLabels) == 0 {
		configuration.AdvertiseNseLabels = getEnv(advertiseNseLabelsEnv, "Advertise labels", false)
	}

	if len(configuration.OutgoingNscLabels) == 0 {
		configuration.OutgoingNscLabels = getEnv(outgoingNscLabelsEnv, "Outgoing labels", false)
	}

	if len(configuration.OutgoingNscMechanism) == 0 {
		configuration.OutgoingNscMechanism = getEnv(outgoingNscMechanismEnv, "Outgoing mechanism type", false)
	}

	if len(configuration.IPAddress) == 0 {
		configuration.IPAddress = getEnv(ipAddressEnv, "Outgoing labels", false)
	}
}
