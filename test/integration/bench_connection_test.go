// +build bench

package nsmd_integration_tests

import (
	"strconv"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestOneTimeConnectionMemif(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneTimeConnection(t, 1, kubetest.DeployVppAgentNSC, kubetest.DeployVppAgentICMP, kubetest.IsVppAgentNsePinged)
}
func TestOneTimeConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneTimeConnection(t, 2, kubetest.DeployNSC, kubetest.DeployICMP, kubetest.IsNsePinged)
}

func TestMovingConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testMovingConnection(t, 2, kubetest.DeployNSC, kubetest.DeployICMP, kubetest.IsNsePinged)
}

func TestMovingConnectionMemif(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testMovingConnection(t, 1, kubetest.DeployVppAgentNSC, kubetest.DeployVppAgentICMP, kubetest.IsVppAgentNsePinged)
}

func TestOneToOneConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneToOneConnection(t, 2, kubetest.DeployNSC, kubetest.DeployICMP, kubetest.IsNsePinged)
}

func TestOneToOneConnectionMemif(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneToOneConnection(t, 1, kubetest.DeployVppAgentNSC, kubetest.DeployVppAgentICMP, kubetest.IsVppAgentNsePinged)
}

func testOneTimeConnection(t *testing.T, nodeCount int, nscDeploy, icmpDeploy kubetest.PodSupplier, nsePing kubetest.NsePinger) {
	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)
	nodes := createNodes(k8s, nodeCount)
	icmpDeploy(k8s, nodes[nodeCount-1].Node, icmpDefaultName, defaultTimeout)

	doneChannel := make(chan nscPingResult, nscCount)
	defer close(doneChannel)

	for count := nscCount; count > 0; count-- {
		go createNscAndPingIcmp(k8s, count, nodes[0].Node, doneChannel, nscDeploy, nsePing)
	}

	for count := nscCount; count > 0; count-- {
		nscPingResult := <-doneChannel
		Expect(nscPingResult.success).To(Equal(true))
	}
}

func testMovingConnection(t *testing.T, nodeCount int, nscDeploy, icmpDeploy kubetest.PodSupplier, pingNse kubetest.NsePinger) {
	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)
	nodes := createNodes(k8s, nodeCount)

	icmpDeploy(k8s, nodes[nodeCount-1].Node, icmpDefaultName, defaultTimeout)
	doneChannel := make(chan nscPingResult, nscCount)
	defer close(doneChannel)

	for testCount := 0; testCount < nscMaxCount; testCount += nscCount {
		for count := nscCount; count > 0; count-- {
			go createNscAndPingIcmp(k8s, count, nodes[0].Node, doneChannel, nscDeploy, pingNse)
		}

		for count := nscCount; count > 0; count-- {
			nscPingResult := <-doneChannel
			Expect(nscPingResult.success).To(Equal(true))
			k8s.DeletePods(nscPingResult.nsc)
		}
	}
}

func testOneToOneConnection(t *testing.T, nodeCount int, nscDeploy, icmpDeploy kubetest.PodSupplier, pingNse kubetest.NsePinger) {
	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)
	nodes := createNodes(k8s, nodeCount)
	doneChannel := make(chan nscPingResult, 1)
	defer close(doneChannel)

	for testCount := 0; testCount < nscMaxCount; testCount += nscCount {
		icmp := icmpDeploy(k8s, nodes[nodeCount-1].Node, icmpDefaultName, defaultTimeout)
		createNscAndPingIcmp(k8s, 1, nodes[0].Node, doneChannel, nscDeploy, pingNse)
		result := <-doneChannel
		Expect(result.success).To(Equal(true))
		k8s.DeletePods(icmp, result.nsc)
	}
}

type nscPingResult struct {
	success bool
	nsc     *v1.Pod
}

func createNodes(k8s *kubetest.K8s, count int) []*kubetest.NodeConf {
	Expect(count > 0 && count < 3).Should(Equal(true))
	nodes, err := kubetest.SetupNodes(k8s, count, defaultTimeout)
	Expect(err).To(BeNil())

	Expect(len(nodes), count)
	return nodes
}

func createNscAndPingIcmp(k8s *kubetest.K8s, id int, node *v1.Node, done chan nscPingResult, nscDeploy kubetest.PodSupplier, pingNse kubetest.NsePinger) {
	nsc := nscDeploy(k8s, node, nscDefaultName+strconv.Itoa(id), defaultTimeout)
	Expect(nsc.Name).To(Equal(nscDefaultName + strconv.Itoa(id)))
	done <- nscPingResult{
		success: pingNse(k8s, nsc),
		nsc:     nsc,
	}
}
