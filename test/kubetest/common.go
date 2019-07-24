package kubetest

import (
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

// DefaultDataplaneVariables - Default variables for dataplane deployment
func DefaultDataplaneVariables(plane string) map[string]string {
	if plane == pods.EnvForwardingPlaneDefault {
		return DefaultPlaneVariablesVPP()
	}
	logrus.Error("Forwarding plane error: Unknown dataplane")
	return nil
}
