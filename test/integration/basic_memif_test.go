// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"testing"
)

func TestSimpleMemifConnection(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.PrepareDefault()

	nodes := nsmd_test_utils.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{})

	nsmd_test_utils.DeployVppAgentICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)
	vppagentNsc := nsmd_test_utils.DeployVppAgentNSC(k8s, nodes[0].Node, "vppagent-nsc", defaultTimeout)
	nsmd_test_utils.IsMemifNsePinged(k8s, vppagentNsc)
}
