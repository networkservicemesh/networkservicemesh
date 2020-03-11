package kubetest

import (
	"time"
)

const (
	artifactFinderWorkerCount = 4
	// PodStartTimeout - Default pod startup time
	PodStartTimeout      = 3 * time.Minute
	podDeleteTimeout     = 15 * time.Second
	podExecTimeout       = 1 * time.Minute
	podGetLogTimeout     = 1 * time.Minute
	accountWaitTimeout   = 1 * time.Minute
	kindControlPlaneNode = "control-plane"

	//NetworkPluginCNIFailure - pattern to check for CNI issue, pattern required to try redeploy pod
	NetworkPluginCNIFailure = "NetworkPlugin cni failed to set up pod"
	envUseIPv6              = "USE_IPV6"
	envUseIPv6Default       = false
)
