package kubetest

import "github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/vppagent"

// DefaultPlaneVariablesKernel - Default variables for Kernel forwarding deployment
func DefaultPlaneVariablesKernel() map[string]string {
	return map[string]string{
		vppagent.DataplaneMetricsCollectorEnabledKey: "false",
	}
}
