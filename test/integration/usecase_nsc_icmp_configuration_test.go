// +build usecase

package nsmd_integration_tests

import (
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSCAndICMPLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, false)
}

func TestNSCAndICMPRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, false)
}

func TestNSCAndICMPWebhookLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, true)
}

func TestNSCAndICMPWebhookRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, true)
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSCAndICMP(t *testing.T, nodesCount int, useWebhook bool) {
	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "nsmd-dataplane", "icmp-responder-nse")
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	nodes_setup := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)

	// Run ICMP on latest node
	_ = nsmd_test_utils.DeployIcmp(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse1", defaultTimeout)

	nscPodNode := nsmd_test_utils.DeployNsc(k8s, nodes_setup[0].Node, "nsc1", defaultTimeout, useWebhook)

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
