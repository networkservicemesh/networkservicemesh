package kubetest

import "github.com/sirupsen/logrus"

// DefaultDataplaneVariables - Default variables for dataplane deployment
func DefaultDataplaneVariables(plane string) map[string]string {
	if plane == "vpp" {
		return DefaultPlaneVariablesVPP()
	}
	logrus.Error("Forwarding plane error: Unknown dataplane")
	return nil
}
