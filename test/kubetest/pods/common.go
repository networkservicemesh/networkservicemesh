package pods

import (
	v1 "k8s.io/api/core/v1"
)

// ForwardingPlane - Wrapper for getting a forwarding plane pod
func ForwardingPlane(name string, node *v1.Node) *v1.Pod {
	return VPPDataplanePod(name, node)
}

// ForwardingPlaneWithConfig - Wrapper for getting a forwarding plane pod
func ForwardingPlaneWithConfig(name string, node *v1.Node, variables map[string]string) *v1.Pod {
	return VPPDataplanePodConfig(name, node, variables)
}

// ForwardingPlaneWithLiveCheck - Wrapper for getting a forwarding plane pod with liveness/readiness probes
func ForwardingPlaneWithLiveCheck(name string, node *v1.Node) *v1.Pod {
	return VPPDataplanePodLiveCheck(name, node)
}
