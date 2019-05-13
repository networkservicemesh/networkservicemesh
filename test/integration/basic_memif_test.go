// +build basic

package nsmd_integration_tests

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/integration/utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
)

func TestSimpleMemifConnection(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	nodes, err := utils.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())
	defer utils.FailLogger(k8s, nodes, t)


	utils.DeployVppAgentICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)
	vppagentNsc := utils.DeployVppAgentNSC(k8s, nodes[0].Node, "vppagent-nsc", defaultTimeout)
	Expect(true, utils.IsVppAgentNsePinged(k8s, vppagentNsc))
}
