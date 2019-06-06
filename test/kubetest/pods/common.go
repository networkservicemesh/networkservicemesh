package pods

import (
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

const (
	// EnvForwardingPlane is the environment variable for configuring the forwarding plane
	EnvForwardingPlane = "FORWARDING_PLANE"
	// EnvForwardingPlaneVPP is the VPP forwarding plane
	EnvForwardingPlaneVPP = "vpp"
	// EnvForwardingPlaneKernel is the Kernel forwarding plane
	EnvForwardingPlaneKernel = "kernel-forwarder"
	// EnvForwardingPlaneDefault is the default forwarding plane
	EnvForwardingPlaneDefault = EnvForwardingPlaneVPP
)

// ForwardingPlane - Wrapper for getting a forwarding plane pod
func ForwardingPlane(name string, node *v1.Node, plane string) *v1.Pod {
	if plane == EnvForwardingPlaneDefault {
		return VPPDataplanePod(name, node)
	} else if plane == EnvForwardingPlaneKernel {
		return KernelDataplanePod(name, node)
	}
	logrus.Error("Forwarding plane error: Unknown dataplane")
	return nil
}

// ForwardingPlaneWithConfig - Wrapper for getting a forwarding plane pod
func ForwardingPlaneWithConfig(name string, node *v1.Node, variables map[string]string, plane string) *v1.Pod {
	if plane == EnvForwardingPlaneDefault {
		return VPPDataplanePodConfig(name, node, variables)
	} else if plane == EnvForwardingPlaneKernel {
		return KernelDataplanePodConfig(name, node, variables)
	}
	logrus.Error("Forwarding plane error: Unknown dataplane")
	return nil
}

// ForwardingPlaneWithLiveCheck - Wrapper for getting a forwarding plane pod with liveness/readiness probes
func ForwardingPlaneWithLiveCheck(name string, node *v1.Node, plane string) *v1.Pod {
	if plane == EnvForwardingPlaneDefault {
		return VPPDataplanePodLiveCheck(name, node)
	} else if plane == EnvForwardingPlaneKernel {
		return KernelDataplanePodLiveCheck(name, node)
	}
	logrus.Error("Forwarding plane error: Unknown dataplane")
	return nil
}
