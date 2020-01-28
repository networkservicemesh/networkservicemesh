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
	// SrcPrivateKey - Source private key
	SrcPrivateKey = "src_private_key"
	// SrcPublicKey - Destination public key
	DstPublicKey = "dst_public_key"
	// DstPrivateKey - Source private key
	DstPrivateKey = "src_private_key"

)
