package vxlan

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
)

const (
	// Mechanism string
	MECHANISM = "VXLAN"

	// Mechanism parameters
	// SrcIP - source IP
	SrcIP = common.SrcIP
	// DstIP - destitiona IP
	DstIP = common.DstIP
	// SrcOriginalIP - original src IP
	SrcOriginalIP = "orig_src_ip"
	// DstExternalIP - external destination ip
	DstExternalIP = "ext_src_ip"
	// VNI - vni
	VNI = "vni"
)
