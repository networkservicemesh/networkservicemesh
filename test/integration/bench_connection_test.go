// +build bench

package nsmd_integration_tests

import (
	"strconv"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/integration/utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
)

func TestOneTimeConnectionMemif(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneTimeConnection(1, utils.DeployVppAgentNSC, utils.DeployVppAgentICMP, utils.IsVppAgentNsePinged)
}
func TestOneTimeConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneTimeConnection(2, utils.DeployNSC, utils.DeployICMP, utils.IsNsePinged)
}

func TestMovingConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testMovingConnection(t, 2, utils.DeployNSC, utils.DeployICMP, utils.IsNsePinged)
}

func TestMovingConnectionMemif(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testMovingConnection(t, 1, utils.DeployVppAgentNSC, utils.DeployVppAgentICMP, utils.IsVppAgentNsePinged)
}

func TestOneToOneConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneToOneConnection(2, utils.DeployNSC, utils.DeployICMP, utils.IsNsePinged)
}

func TestOneToOneConnectionMemif(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneToOneConnection(1, utils.DeployVppAgentNSC, utils.DeployVppAgentICMP, utils.IsVppAgentNsePinged)
}

func testOneTimeConnection(nodeCount int, nscDeploy, icmpDeploy utils.PodSupplier, nsePing utils.NsePinger) {
	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

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

func testMovingConnection(t *testing.T, nodeCount int, nscDeploy, icmpDeploy utils.PodSupplier, pingNse utils.NsePinger) {
	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	nodes := createNodes(k8s, nodeCount)
	defer utils.FailLogger(k8s, nodes, t)

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

func testOneToOneConnection(nodeCount int, nscDeploy, icmpDeploy utils.PodSupplier, pingNse utils.NsePinger) {
	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

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

func createNodes(k8s *kube_testing.K8s, count int) []*utils.NodeConf {
	Expect(count > 0 && count < 3).Should(Equal(true))
	nodes := utils.SetupNodes(k8s, count, defaultTimeout)
	Expect(len(nodes), count)
	return nodes
}

func createNscAndPingIcmp(k8s *kube_testing.K8s, id int, node *v1.Node, done chan nscPingResult, nscDeploy utils.PodSupplier, pingNse utils.NsePinger) {
	nsc := nscDeploy(k8s, node, nscDefaultName+strconv.Itoa(id), defaultTimeout)
	Expect(nsc.Name).To(Equal(nscDefaultName + strconv.Itoa(id)))
	done <- nscPingResult{
		success: pingNse(k8s, nsc),
		nsc:     nsc,
	}
}
