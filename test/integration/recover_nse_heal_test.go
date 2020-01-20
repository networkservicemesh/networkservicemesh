// +build recover_suite

package integration

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestNSEHealLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testNSEHeal(
		&testNSEHealParameters{t: t,
			nodesCount: 1,
			affinity: map[string]int{
				"icmp-responder-nse-1": 0,
				"icmp-responder-nse-2": 0,
			},
			fixture:     kubetest.DefaultTestingPodFixture(g),
			clearOption: kubetest.ReuseNSMResources,
		},
	)
}

func TestNSEHealLocalToRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testNSEHeal(
		&testNSEHealParameters{t: t,
			nodesCount: 2,
			affinity: map[string]int{
				"icmp-responder-nse-1": 0,
				"icmp-responder-nse-2": 1,
			},
			fixture:     kubetest.DefaultTestingPodFixture(g),
			clearOption: kubetest.ReuseNSMResources,
		},
	)
}

func TestNSEHealRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testNSEHeal(
		&testNSEHealParameters{t: t,
			nodesCount: 2,
			affinity: map[string]int{
				"icmp-responder-nse-1": 1,
				"icmp-responder-nse-2": 1,
			},
			fixture:     kubetest.DefaultTestingPodFixture(g),
			clearOption: kubetest.ReuseNSMResources,
		},
	)
}

func TestNSEHealLocalVpp(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(
		&testNSEHealParameters{t: t,
			nodesCount: 1,
			affinity: map[string]int{
				"icmp-responder-nse-1": 0,
				"icmp-responder-nse-2": 0,
			},
			fixture:     kubetest.DefaultTestingPodFixture(g),
			clearOption: kubetest.ReuseNSMResources,
		},
	)
}

func TestNSEHealToLocalVpp(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(
		&testNSEHealParameters{t: t,
			nodesCount: 2,
			affinity: map[string]int{
				"icmp-responder-nse-1": 1,
				"icmp-responder-nse-2": 0,
			},
			fixture:     kubetest.VppAgentTestingPodFixture(g),
			clearOption: kubetest.ReuseNSMResources,
		},
	)
}

func TestNSEHealToRemoteVpp(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(
		&testNSEHealParameters{t: t,
			nodesCount: 2,
			affinity: map[string]int{
				"icmp-responder-nse-1": 0,
				"icmp-responder-nse-2": 1,
			},
			fixture:     kubetest.VppAgentTestingPodFixture(g),
			clearOption: kubetest.ReuseNSMResources,
		},
	)
}

func TestNSEHealRemoteVpp(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testNSEHeal(
		&testNSEHealParameters{t: t,
			nodesCount: 2,
			affinity: map[string]int{
				"icmp-responder-nse-1": 1,
				"icmp-responder-nse-2": 1,
			},
			fixture:     kubetest.VppAgentTestingPodFixture(g),
			clearOption: kubetest.ReuseNSMResources,
		},
	)
}

func TestClosingNSEHealRemoteToLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	affinity := map[string]int{
		"icmp-responder-nse-1": 1,
		"icmp-responder-nse-2": 0,
	}
	fixture := kubetest.DefaultTestingPodFixture(g)

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	g.Expect(err).To(BeNil())
	defer k8s.Cleanup(t)

	// Deploy open tracing to see what happening.
	nodesSetup, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.SaveTestArtifacts(t)

	// Run ICMP
	node := affinity["icmp-responder-nse-1"]
	nse1 := fixture.DeployNse(k8s, nodesSetup[node].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout)
	fixture.CheckNsc(k8s, nscPodNode)

	// Delete NSE
	k8s.DeletePods(nse1)
	// Wait for DST Heal
	logrus.Infof("Waiting for connection starts recovering...")
	k8s.WaitLogsContains(nodesSetup[0].Nsmd, "nsmd", "Starting DST Heal...", defaultTimeout)
	// Delete NSC
	k8s.DeletePods(nscPodNode)

	// Run NSE and NSC
	node = affinity["icmp-responder-nse-2"]
	nse1 = fixture.DeployNse(k8s, nodesSetup[node].Node, "icmp-responder-nse-1", defaultTimeout)
	nscPodNode = fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout)

	fixture.CheckNsc(k8s, nscPodNode)
}
