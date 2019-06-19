package kubetest

import "github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"

// DefaultPlaneVariablesKernel - Default variables for Kernel forwarding deployment
func DefaultPlaneVariablesKernel() map[string]string {
	return map[string]string{
		common.DataplaneMetricsEnabledKey: "false",
	}
}
