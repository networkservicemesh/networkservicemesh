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
	ncmAgentName   = "vppagent-nsc"
	icmpAgentName  = "vppagent-icmp-responder"
	ipAddressParam = "IP_ADDRESS"
	threadsCount   = 10
)

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

func createAndPingIcmp(k8s *kube_testing.K8s, id int, node *v1.Node, vppIcmpAgent *v1.Pod, done chan bool) {
	envNsc := map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   "icmp-responder",
	}
	vppAgentNsc := k8s.CreatePod(pods.VppagentNSC(ncmAgentName+strconv.Itoa(id), node, envNsc))
	Expect(vppAgentNsc.Name).To(Equal(ncmAgentName + strconv.Itoa(id)))
	result := false
	for attempts := 30; attempts > 0; <-time.Tick(300 * time.Millisecond) {
		response, _ := ping(k8s, vppAgentNsc, vppIcmpAgent)
		if !strings.Contains(response, "100% packet loss") {
			result = true
			logrus.Info("Ping successful")
			break
		}
		attempts--
	}
	done <- result
}

func TestBenchMemifConnection(t *testing.T) {

	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.PrepareDefault()

	node := createNode(k8s)
	envIcmp := map[string]string{
		"ADVERTISE_NSE_NAME":   "icmp-responder",
		"ADVERTISE_NSE_LABELS": "app=icmp",
		"IP_ADDRESS":           "10.20.1.0/24",
	}
	vppAgentIcmp := k8s.CreatePod(pods.VppagentICMPResponderPod(icmpAgentName, node, envIcmp))
	Expect(vppAgentIcmp.Name).To(Equal(icmpAgentName))

	doneChannel := make(chan bool, threadsCount)
	for count := threadsCount; count >= 0; count-- {
		go createAndPingIcmp(k8s, count, node, vppAgentIcmp, doneChannel)
	}

	for count := threadsCount; count >= 0; count-- {
		Expect(<-doneChannel).To(Equal(true))
	}
}
