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
	SrcOriginalIP = common.SrcOriginalIP
	// DstExternalIP - external destination ip
	DstExternalIP = common.DstExternalIP
	// VNI - vni
	VNI = "vni"
)
