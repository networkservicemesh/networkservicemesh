package pods

import (
	v1 "k8s.io/api/core/v1"
	"github.com/sirupsen/logrus"
)

// ForwardingPlane - Wrapper for getting a forwarding plane pod
func ForwardingPlane(name string, node *v1.Node, plane string) *v1.Pod {
	if plane == "vpp" {
		return VPPDataplanePod(name, node)
	}
	logrus.Error("Forwarding plane error: Unknown dataplane")
	return nil
}

// ForwardingPlaneWithConfig - Wrapper for getting a forwarding plane pod
func ForwardingPlaneWithConfig(name string, node *v1.Node, variables map[string]string, plane string) *v1.Pod {
	if plane == "vpp" {
		return VPPDataplanePodConfig(name, node, variables)
	}
	logrus.Error("Forwarding plane error: Unknown dataplane")
	return nil
}

// ForwardingPlaneWithLiveCheck - Wrapper for getting a forwarding plane pod with liveness/readiness probes
func ForwardingPlaneWithLiveCheck(name string, node *v1.Node, plane string) *v1.Pod {
	if plane == "vpp" {
		return VPPDataplanePodLiveCheck(name, node)
	}
	logrus.Error("Forwarding plane error: Unknown dataplane")
	return nil
}
