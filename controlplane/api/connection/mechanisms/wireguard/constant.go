package wireguard

import (
"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
)

const (
	// Mechanism string
	MECHANISM = "WIREGUARD"

	// Mechanism parameters
	// SrcIP - source IP
	SrcIP = common.SrcIP
	// DstIP - destitiona IP
	DstIP = common.DstIP
	// SrcOriginalIP - original src IP
	SrcOriginalIP = "orig_src_ip"
	// DstExternalIP - external destination ip
	DstExternalIP = "ext_src_ip"
	// SrcPublicKey - Source public key
	SrcPublicKey = "src_public_key"
	// SrcPublicKey - Destination public key
	DstPublicKey = "dst_public_key"

)

