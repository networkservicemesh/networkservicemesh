package vxlan

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
)

const (
	// Mechanism string
	MECHANISM = "VXLAN"

	// Mechanism parameters
	SrcIP = common.SrcIP
	DstIP = common.DstIP
	VNI   = "vni"
)
