package mechanism_kernel

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/mechanisms/mechanism_kernel/mechanism_kernel_tap"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/mechanisms/mechanism_kernel/mechanism_kernel_vethpair"
)

func NewServer(baseDir string) networkservice.NetworkServiceServer {
	if useVHostNet() {
		return mechanism_kernel_tap.NewServer(baseDir)
	}
	return mechanism_kernel_vethpair.NewServer(baseDir)
}
