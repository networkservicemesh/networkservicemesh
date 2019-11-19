// +build usecase

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestBrokenConnWithLocalNSE(t *testing.T) {
	testBrokenConnection(t, false)
}

func TestBrokenConnWithRemoteNSE(t *testing.T) {
	testBrokenConnection(t, true)
}

func testBrokenConnection(t *testing.T, isRemote bool) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	g.Expect(err).To(BeNil())

	defer k8s.Cleanup()

	var nodesCount int
	if isRemote {
		nodesCount = 2
	} else {
		nodesCount = 1
	}

	nodesSetup, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	g.Expect(err).To(BeNil())

	defer kubetest.MakeLogsSnapshot(k8s, t)

	nsesInitial, err := k8s.GetNSEs()
	g.Expect(err).To(BeNil())
	g.Expect(len(nsesInitial)).To(Equal(0))

	nse := kubetest.DeployICMP(k8s, nodesSetup[0].Node, "icmp-responder-nse-1", defaultTimeout)
	nsc := kubetest.DeployNSC(k8s, nodesSetup[nodesCount-1].Node, "nsc-1", defaultTimeout)
	kubetest.CheckNSC(k8s, nsc)

	nsesBefore, err := k8s.GetNSEs()
	g.Expect(err).To(BeNil())
	g.Expect(len(nsesBefore)).To(Equal(1))

	k8s.DeletePods(nse)

	k8s.WaitLogsContains(nodesSetup[0].Nsmd, "nsmd", "ClientConnection dst state is down. calling Heal", defaultTimeout)

	k8s.DeletePods(nsc)

	k8s.WaitLogsContains(nodesSetup[0].Nsmd, "nsmd", "ClientConnectionDeleted:", defaultTimeout)

	nsesAfter, err := k8s.GetNSEs()
	g.Expect(err).To(BeNil())
	g.Expect(len(nsesAfter)).To(Equal(0))
}
