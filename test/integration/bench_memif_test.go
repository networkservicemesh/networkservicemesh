// +build bench

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"strconv"
	"testing"
)

const (
	nscAgentName  = "vppagent-nsc"
	icmpAgentName = "icmp-responder"
	nscCount      = 5
	nscMaxCount   = 10
)

func TestBenchMemifOneTimeConnecting(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.PrepareDefault()

	node := createNode(k8s)
	nsmd_test_utils.DeployVppAgentICMP(k8s, node, icmpAgentName, defaultTimeout)

	doneChannel := make(chan nscPingResult, nscCount)
	defer close(doneChannel)

	for count := nscCount; count >= 0; count-- {
		go createNscAndPingIcmp(k8s, count, node, doneChannel)
	}

	for count := nscCount; count >= 0; count-- {
		nscPingResult := <-doneChannel
		Expect(nscPingResult.success).To(Equal(true))
	}
}

func TestBenchMemifMovingConnection(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.PrepareDefault()

	node := createNode(k8s)

	nsmd_test_utils.DeployVppAgentICMP(k8s, node, icmpAgentName, defaultTimeout)
	doneChannel := make(chan nscPingResult, nscCount)
	defer close(doneChannel)

	for testCount := 0; testCount < nscMaxCount; testCount += nscCount {
		for count := nscCount; count >= 0; count-- {
			go createNscAndPingIcmp(k8s, count, node, doneChannel)
		}

		for count := nscCount; count >= 0; count-- {
			nscPingResult := <-doneChannel
			Expect(nscPingResult.success).To(Equal(true))
			k8s.DeletePods(nscPingResult.nsc)
		}
	}
}

func TestBenchMemifPerToPer(t *testing.T) {
	RegisterTestingT(t)
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.PrepareDefault()
	node := createNode(k8s)
	doneChannel := make(chan nscPingResult, 1)
	defer close(doneChannel)

	for testCount := 0; testCount < nscMaxCount; testCount += nscCount {
		icmp := nsmd_test_utils.DeployVppAgentICMP(k8s, node, icmpAgentName, defaultTimeout)
		createNscAndPingIcmp(k8s, 1, node, doneChannel)
		result := <-doneChannel
		Expect(result.success).To(Equal(true))
		k8s.DeletePods(icmp, result.nsc)
	}
}

type nscPingResult struct {
	success bool
	nsc     *v1.Pod
}

func createNode(k8s *kube_testing.K8s) *v1.Node {
	nodes := nsmd_test_utils.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{})
	Expect(len(nodes), 1)
	return nodes[0].Node
}

func createNscAndPingIcmp(k8s *kube_testing.K8s, id int, node *v1.Node, done chan nscPingResult) {
	vppAgentNsc := nsmd_test_utils.DeployVppAgentNSC(k8s, node, nscAgentName+strconv.Itoa(id), defaultTimeout)
	Expect(vppAgentNsc.Name).To(Equal(nscAgentName + strconv.Itoa(id)))
	done <- nscPingResult{
		success: nsmd_test_utils.IsMemifNsePinged(k8s, vppAgentNsc),
		nsc:     vppAgentNsc,
	}
}
