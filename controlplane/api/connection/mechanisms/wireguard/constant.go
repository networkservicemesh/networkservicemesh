// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package wireguard - constants and helper methods for Wireguard remote mechanism
package wireguard

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
)

const (
	// MECHANISM type string
	MECHANISM = "WIREGUARD"

	// BasePort - Wireguard base port
	BasePort = 51820

	// Mechanism parameters

	// SrcIP - source IP
	SrcIP = common.SrcIP
	// DstIP - destitiona IP
	DstIP = common.DstIP
	// SrcOriginalIP - original src IP
	SrcOriginalIP = "orig_src_ip"
	// DstExternalIP - external destination ip
	DstExternalIP = "ext_src_ip"
	// SrcPort - Source interface listening port
	SrcPort = "src_port"
	// DstPort - Destination interface listening port
	DstPort = "dst_port"
	// SrcPublicKey - Source public key
	SrcPublicKey = "src_public_key"
	// SrcPrivateKey - Source private key
	SrcPrivateKey = "src_private_key"
	// DstPublicKey - Destination public key
	DstPublicKey = "dst_public_key"
	// DstPrivateKey - Source private key
	DstPrivateKey = "dst_private_key"
)
