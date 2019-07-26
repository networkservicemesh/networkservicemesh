// +build recover

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"strings"
	"testing"
	"time"
)

func TestNSCDiesSingleNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, true, 1)
}

func TestNSEDiesSingleNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, false, 1)
}

func TestNSCDiesMultiNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testDie(t, true, 2)
}
func TestNSEDiesMultiNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testDie(t, false, 2)
}

func testDie(t *testing.T, killSrc bool, nodesCount int) {
	Expect(nodesCount > 0).Should(BeTrue())

	k8s, err := kubetest.NewK8s(true)

	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodes, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, kubetest.NoHealNSMgrPodConfig(k8s), k8s.GetK8sNamespace())

	defer kubetest.ShowLogs(k8s, t)
	Expect(err).To(BeNil())

	icmp := kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsc := kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

	ipResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ip", "addr")
	Expect(err).To(BeNil())
	Expect(errOut).To(Equal(""))
	Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

	ipResponse, errOut, err = k8s.Exec(icmp, icmp.Spec.Containers[0].Name, "ip", "addr")
	Expect(err).To(BeNil())
	Expect(errOut).To(Equal(""))
	Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

	pingResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ping", "172.16.1.2", "-A", "-c", "5")
	Expect(err).To(BeNil())
	Expect(strings.Contains(pingResponse, "5 packets transmitted, 5 packets received, 0% packet loss")).To(Equal(true))
	logrus.Printf("NSC Ping is success:%s", pingResponse)

	var podToKill *v1.Pod
	var podToCheck *v1.Pod
	if killSrc {
		podToKill = nsc
		podToCheck = icmp
	} else {
		podToKill = icmp
		podToCheck = nsc
	}

	k8s.DeletePods(podToKill)
	success := false
	for attempt := 0; attempt < 20; <-time.After(300 * time.Millisecond) {
		attempt++
		ipResponse, errOut, err = k8s.Exec(podToCheck, podToCheck.Spec.Containers[0].Name, "ip", "addr")
		if !strings.Contains(ipResponse, "nsm") {
			success = true
			break
		}
	}
	Expect(success).To(Equal(true))

}
