// +build basic

package integration

import (
	"strconv"
	"testing"

	v1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"

	interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestSimpleMemifConnectionL2(t *testing.T) {
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
	defer k8s.SaveTestArtifacts(t)

	// L2 memif mode is used by default
	kubetest.DeployVppAgentICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)

	vppagentNsc := kubetest.DeployVppAgentNSC(k8s, nodes[0].Node, "vppagent-nsc", defaultTimeout)
	g.Expect(kubetest.IsVppAgentNsePinged(k8s, vppagentNsc)).To(BeTrue())
}

func TestSimpleMemifConnectionL3(t *testing.T) {
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
	defer k8s.SaveTestArtifacts(t)

	// Set L3 memif mode
	additionEnv := v1.EnvVar{
		Name:  "MEMIF_MODE",
		Value: strconv.Itoa(int(interfaces.MemifLink_IP)),
	}
	kubetest.DeployVppAgentICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout, additionEnv)

	vppagentNsc := kubetest.DeployVppAgentNSC(k8s, nodes[0].Node, "vppagent-nsc", defaultTimeout, additionEnv)
	g.Expect(kubetest.IsVppAgentNsePinged(k8s, vppagentNsc)).To(BeTrue())
}
