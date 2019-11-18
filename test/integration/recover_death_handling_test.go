// +build recover

package nsmd_integration_tests

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestNSCDiesSingleNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, true, 1)
}

func TestNSEDiesSingleNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, false, 1)
}

func TestNSCDiesMultiNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testDie(t, true, 2)
}

func TestNSEDiesMultiNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testDie(t, false, 2)
}

func testDie(t *testing.T, killSrc bool, nodesCount int) {
	g := NewWithT(t)

	g.Expect(nodesCount > 0).Should(BeTrue())

	k8s, err := kubetest.NewK8s(g, kubetest.DefaultClear)

	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodes, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, kubetest.NoHealNSMgrPodConfig(k8s), k8s.GetK8sNamespace())

	defer k8s.ProcessArtifacts(t)
	g.Expect(err).To(BeNil())

	icmp := kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsc := kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

	ipResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ip", "addr")
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	g.Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

	ipResponse, errOut, err = k8s.Exec(icmp, icmp.Spec.Containers[0].Name, "ip", "addr")
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	g.Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

	pingResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ping", "172.16.1.2", "-A", "-c", "5")
	g.Expect(err).To(BeNil())
	g.Expect(strings.Contains(pingResponse, "100% packet loss")).To(Equal(false))
	logrus.Printf("NSC Ping is success:%s", pingResponse)

	var podToKill *v1.Pod
	var podToCheck *v1.Pod
	var nsmdPodToCheck *v1.Pod
	if killSrc {
		podToKill = nsc
		podToCheck = icmp
		nsmdPodToCheck = nodes[nodesCount-1].Nsmd
	} else {
		podToKill = icmp
		podToCheck = nsc
		nsmdPodToCheck = nodes[0].Nsmd
	}

	k8s.DeletePods(podToKill)
	k8s.WaitLogsContains(nsmdPodToCheck, "nsmd", "Cross connection successfully closed on forwarder", defaultTimeout)

	ipResponse, errOut, err = k8s.Exec(podToCheck, podToCheck.Spec.Containers[0].Name, "ip", "addr")
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	g.Expect(ipResponse).ShouldNot(ContainSubstring("nsm"))
}
