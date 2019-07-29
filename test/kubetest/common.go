package kubetest

import (
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

// DefaultDataplaneVariables - Default variables for dataplane deployment
func DefaultDataplaneVariables(plane string) map[string]string {
	if plane == pods.EnvForwardingPlaneDefault {
		return DefaultPlaneVariablesVPP()
	} else if plane == pods.EnvForwardingPlaneKernel {
		return DefaultPlaneVariablesKernel()
	}
	logrus.Error("Forwarding plane error: Unknown dataplane")
	return nil
}
