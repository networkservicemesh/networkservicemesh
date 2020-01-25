// +build basic

package integration

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestSimpleMemifConnection(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)

	kubetest.DeployVppAgentICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)
	vppagentNsc := kubetest.DeployVppAgentNSC(k8s, nodes[0].Node, "vppagent-nsc", defaultTimeout)
	g.Expect(true, kubetest.IsVppAgentNsePinged(k8s, vppagentNsc))
}
