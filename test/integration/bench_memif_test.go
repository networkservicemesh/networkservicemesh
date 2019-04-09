package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	nscAgentName   = "vppagent-nsc"
	icmpAgentName  = "vppagent-icmp-responder"
	ipAddressParam = "IP_ADDRESS"
	nscCount       = 5
	nscMaxCount    = 15
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
	vppAgentIcmp := k8s.CreatePod(pods.VppagentICMPResponderPod(icmpAgentName, node, icmpEnv()))
	Expect(vppAgentIcmp.Name).To(Equal(icmpAgentName))

	doneChannel := make(chan nscPingResult, nscCount)
	defer close(doneChannel)

	for count := nscMaxCount; count >= 0; count-- {
		go createNscAndPingIcmp(k8s, count, node, vppAgentIcmp, doneChannel)
	}

	for count := nscMaxCount; count >= 0; count-- {
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

	vppAgentIcmp := k8s.CreatePod(pods.VppagentICMPResponderPod(icmpAgentName, node, nscEnv()))
	Expect(vppAgentIcmp.Name).To(Equal(icmpAgentName))

	doneChannel := make(chan nscPingResult, nscCount)
	defer close(doneChannel)

	for testCount := 0; testCount < nscMaxCount; testCount += nscCount {
		for count := nscCount; count >= 0; count-- {
			go createNscAndPingIcmp(k8s, count, node, vppAgentIcmp, doneChannel)
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
		vppAgentIcmp := k8s.CreatePod(pods.VppagentICMPResponderPod(icmpAgentName, node, nscEnv()))
		Expect(vppAgentIcmp.Name).To(Equal(icmpAgentName))
		createNscAndPingIcmp(k8s, 1, node, vppAgentIcmp, doneChannel)
		result := <-doneChannel
		Expect(result.success).To(Equal(true))
		k8s.DeletePods(vppAgentIcmp, result.nsc)
	}
}

type nscPingResult struct {
	success bool
	nsc     *v1.Pod
}

func icmpEnv() map[string]string {
	return map[string]string{
		"ADVERTISE_NSE_NAME":   "icmp-responder",
		"ADVERTISE_NSE_LABELS": "app=icmp",
		"IP_ADDRESS":           "10.20.1.0/24",
	}
}

func nscEnv() map[string]string {
	return map[string]string{
		"ADVERTISE_NSE_NAME":   "icmp-responder",
		"ADVERTISE_NSE_LABELS": "app=icmp",
		"IP_ADDRESS":           "10.20.1.0/24",
	}
}

func createNode(k8s *kube_testing.K8s) *v1.Node {
	nodes := nsmd_test_utils.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{})
	Expect(len(nodes), 1)
	return nodes[0].Node
}

func ping(k8s *kube_testing.K8s, from *v1.Pod, to *v1.Pod) (string, error) {
	var ip string
	for _, val := range to.Spec.Containers[0].Env {
		if val.Name == ipAddressParam {
			ip = val.Value
			break
		}
	}
	Expect(ip).ShouldNot(BeNil())
	response, _, err := k8s.Exec(from, from.Spec.Containers[0].Name, "vppctl", "ping", ip)
	return response, err
}

func createNscAndPingIcmp(k8s *kube_testing.K8s, id int, node *v1.Node, vppIcmpAgent *v1.Pod, done chan nscPingResult) {
	vppAgentNsc := k8s.CreatePod(pods.VppagentNSC(nscAgentName+strconv.Itoa(id), node, nscEnv()))
	Expect(vppAgentNsc.Name).To(Equal(nscAgentName + strconv.Itoa(id)))
	result := false
	for attempts := 10; attempts > 0; <-time.Tick(1000 * time.Millisecond) {
		response, _ := ping(k8s, vppAgentNsc, vppIcmpAgent)
		if !strings.Contains(response, "100% packet loss") {
			result = true
			logrus.Info("Ping successful")
			break
		}
		attempts--
	}
	done <- nscPingResult{
		success: result,
		nsc:     vppAgentNsc}
}
