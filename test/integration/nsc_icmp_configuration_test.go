package nsmd_integration_tests

import (
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func createNscPod(node *v1.Node) *v1.Pod {
	return pods.NSCPod("nsc1", node,
		map[string]string{
			"OUTGOING_NSC_LABELS": "app=icmp",
			"OUTGOING_NSC_NAME":   "icmp-responder",
		},
	)
}

func TestNSCAndICMPLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, createNscPod)
}

func TestNSCAndICMPRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, createNscPod)
}

func TestNSCAndICMPWebhookLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, func(node *v1.Node) *v1.Pod {
		return pods.NSCPodWebhook("nsc1", node)
	})
}

func TestNSCAndICMPWebhookRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, func(node *v1.Node) *v1.Pod {
		return pods.NSCPodWebhook("nsc1", node)
	})
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSCAndICMP(t *testing.T, nodesCount int, nscPodFactory func(*v1.Node) *v1.Pod) {
	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "nsmd-dataplane", "icmp-responder-nse")
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	nodes_setup := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)

	// Run ICMP on latest node
	_ = nsmd_test_utils.DeployIcmp(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse1", defaultTimeout)

	nscPodNode := nsmd_test_utils.DeployNsc(k8s, nodes_setup[0].Node, "nsc1", defaultTimeout)

	var nscInfo *nsmd_test_utils.NSCCheckInfo

	failures := InterceptGomegaFailures(func() {
		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	if len(failures) > 0 {
		logrus.Errorf("Failues: %v", failures)
		nsmd_test_utils.PrintLogs(k8s, nodes_setup)
		nscInfo.PrintLogs()

		t.Fail()
	}
}
