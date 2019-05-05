// +build bench

package nsmd_integration_tests

import (
	"strconv"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
)

func TestOneTimeConnectionMemif(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneTimeConnection(1, nsmd_test_utils.DeployVppAgentNSC, nsmd_test_utils.DeployVppAgentICMP, nsmd_test_utils.IsVppAgentNsePinged)
}
func TestOneTimeConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneTimeConnection(2, nsmd_test_utils.DeployNSC, nsmd_test_utils.DeployICMP, nsmd_test_utils.IsNsePinged)
}

func TestMovingConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testMovingConnection(2, nsmd_test_utils.DeployNSC, nsmd_test_utils.DeployICMP, nsmd_test_utils.IsNsePinged)
}

func TestMovingConnectionMemif(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testMovingConnection(1, nsmd_test_utils.DeployVppAgentNSC, nsmd_test_utils.DeployVppAgentICMP, nsmd_test_utils.IsVppAgentNsePinged)
}

func TestOneToOneConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneToOneConnection(2, nsmd_test_utils.DeployNSC, nsmd_test_utils.DeployICMP, nsmd_test_utils.IsNsePinged)
}

func TestOneToOneConnectionMemif(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testOneToOneConnection(1, nsmd_test_utils.DeployVppAgentNSC, nsmd_test_utils.DeployVppAgentICMP, nsmd_test_utils.IsVppAgentNsePinged)
}

func testOneTimeConnection(nodeCount int, nscDeploy, icmpDeploy nsmd_test_utils.PodSupplier, nsePing nsmd_test_utils.NsePinger) {
	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	nodes := createNodes(k8s, nodeCount)
	icmpDeploy(k8s, nodes[nodeCount-1], icmpDefaultName, defaultTimeout)

	doneChannel := make(chan nscPingResult, nscCount)
	defer close(doneChannel)

	for count := nscCount; count > 0; count-- {
		go createNscAndPingIcmp(k8s, count, nodes[0], doneChannel, nscDeploy, nsePing)
	}

	for count := nscCount; count > 0; count-- {
		nscPingResult := <-doneChannel
		Expect(nscPingResult.success).To(Equal(true))
	}
}

func testMovingConnection(nodeCount int, nscDeploy, icmpDeploy nsmd_test_utils.PodSupplier, pingNse nsmd_test_utils.NsePinger) {
	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	nodes := createNodes(k8s, nodeCount)

	icmpDeploy(k8s, nodes[nodeCount-1], icmpDefaultName, defaultTimeout)
	doneChannel := make(chan nscPingResult, nscCount)
	defer close(doneChannel)

	for testCount := 0; testCount < nscMaxCount; testCount += nscCount {
		for count := nscCount; count > 0; count-- {
			go createNscAndPingIcmp(k8s, count, nodes[0], doneChannel, nscDeploy, pingNse)
		}

		for count := nscCount; count > 0; count-- {
			nscPingResult := <-doneChannel
			Expect(nscPingResult.success).To(Equal(true))
			k8s.DeletePods(nscPingResult.nsc)
		}
	}
}

func testOneToOneConnection(nodeCount int, nscDeploy, icmpDeploy nsmd_test_utils.PodSupplier, pingNse nsmd_test_utils.NsePinger) {
	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	nodes := createNodes(k8s, nodeCount)
	doneChannel := make(chan nscPingResult, 1)
	defer close(doneChannel)

	for testCount := 0; testCount < nscMaxCount; testCount += nscCount {
		icmp := icmpDeploy(k8s, nodes[nodeCount-1], icmpDefaultName, defaultTimeout)
		createNscAndPingIcmp(k8s, 1, nodes[0], doneChannel, nscDeploy, pingNse)
		result := <-doneChannel
		Expect(result.success).To(Equal(true))
		k8s.DeletePods(icmp, result.nsc)
	}
}

type nscPingResult struct {
	success bool
	nsc     *v1.Pod
}

func createNodes(k8s *kube_testing.K8s, count int) []*v1.Node {
	Expect(count > 0 && count < 3).Should(Equal(true))
	nodes := nsmd_test_utils.SetupNodesConfig(k8s, count, defaultTimeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
	Expect(len(nodes), count)
	result := make([]*v1.Node, count)
	for i := 0; i < count; i++ {
		result[i] = nodes[i].Node
	}
	return result
}

func createNscAndPingIcmp(k8s *kube_testing.K8s, id int, node *v1.Node, done chan nscPingResult, nscDeploy nsmd_test_utils.PodSupplier, pingNse nsmd_test_utils.NsePinger) {
	nsc := nscDeploy(k8s, node, nscDefaultName+strconv.Itoa(id), defaultTimeout)
	Expect(nsc.Name).To(Equal(nscDefaultName + strconv.Itoa(id)))
	done <- nscPingResult{
		success: pingNse(k8s, nsc),
		nsc:     nsc,
	}
}
