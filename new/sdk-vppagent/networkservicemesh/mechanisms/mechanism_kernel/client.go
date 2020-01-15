package mechanism_kernel

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/mechanisms/mechanism_kernel/mechanism_kernel_tap"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/mechanisms/mechanism_kernel/mechanism_kernel_vethpair"
)

func NewClient() networkservice.NetworkServiceClient {
	if useVHostNet() {
		return mechanism_kernel_tap.NewClient()
	}
	return mechanism_kernel_vethpair.NewClient()
}
