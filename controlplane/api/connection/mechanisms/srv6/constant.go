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

// Package srv6 - parameters keys of SRV6 remote mechanism
package srv6

const (
	// MECHANISM string
	MECHANISM = "SRV6"

	// Mechanism parameters

	// SrcHostIP -  src localsid of mgmt interface
	SrcHostIP = "src_host_ip"
	// DstHostIP -  dst localsid of mgmt interface
	DstHostIP = "dst_host_ip"
	// SrcBSID -  src BSID
	SrcBSID = "src_bsid"
	// DstBSID - dst BSID
	DstBSID = "dst_bsid"
	// SrcLocalSID -  src LocalSID
	SrcLocalSID = "src_localsid"
	// DstLocalSID - dst LocalSID
	DstLocalSID = "dst_localsid"
	// SrcHostLocalSID -  src host unique LocalSID
	SrcHostLocalSID = "src_host_localsid"
	// DstHostLocalSID - dst host unique LocalSID
	DstHostLocalSID = "dst_host_localsid"
	// SrcHardwareAddress -  src hw address
	SrcHardwareAddress = "src_hw_addr"
	// DstHardwareAddress - dst hw address
	DstHardwareAddress = "dst_hw_addr"
)
