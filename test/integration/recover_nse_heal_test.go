// +build recover

package nsmd_integration_tests

import (
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSEHealLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 1)
}

func TestNSEHealRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 2)
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSEHeal(t *testing.T, nodesCount int) {
	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "nsmd-dataplane", "icmp-responder-nse", "jaeger")
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	// Deploy open tracing to see what happening.
	nodes_setup := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)

	// Run ICMP on latest node
	nse1 := nsmd_test_utils.DeployIcmp(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse1", defaultTimeout)

	nscPodNode := nsmd_test_utils.DeployNsc(k8s, nodes_setup[0].Node, "nsc1", defaultTimeout, false)
	var nscInfo *nsmd_test_utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	printErrors(failures, k8s, nodes_setup, nscInfo, t)

	// Since all is fine now, we need to add new ICMP responder and delete previous one.
	_ = nsmd_test_utils.DeployIcmp(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse2", defaultTimeout)

	logrus.Infof("Delete first NSE")
	k8s.DeletePods(nse1)

	logrus.Infof("Waiting for connection recovery...")
	failures = InterceptGomegaFailures(func() {
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	})
	if len(failures) > 0 {
		printErrors(failures, k8s, nodes_setup, nscInfo, t)
	}

	if len(nodes_setup) > 1 {
		l2, err := k8s.GetLogs(nodes_setup[1].Nsmd, "nsmd")
		Expect(err).To(BeNil())
		if strings.Contains(l2, "Dataplane1 request failed:") {
			logrus.Infof("Dataplane first attempt was failed: %v", l2)
		}
	}

	failures = InterceptGomegaFailures(func() {
		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	printErrors(failures, k8s, nodes_setup, nscInfo, t)
}

func printErrors(failures []string, k8s *kube_testing.K8s, nodes_setup []*nsmd_test_utils.NodeConf, nscInfo *nsmd_test_utils.NSCCheckInfo, t *testing.T) {
	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)
		nsmd_test_utils.PrintLogs(k8s, nodes_setup)
		nscInfo.PrintLogs()

		t.Fail()
	}
}
