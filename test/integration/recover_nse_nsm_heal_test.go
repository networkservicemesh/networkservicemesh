// +build recover

package nsmd_integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSMHealLocalDieNSMD(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup := nsmd_test_utils.SetupNodes(k8s, 2, defaultTimeout)

	// Run ICMP on latest node
	icmpPod := nsmd_test_utils.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())

	nscPodNode := nsmd_test_utils.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *nsmd_test_utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	logrus.Infof("Delete Local NSMD")
	k8s.DeletePods(nodes_setup[0].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[1].Nsmd, "nsmd", "NSM: Remote opened connection is not monitored and put into Healing state", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("%s-recovered", nodes_setup[0].Nsmd.Name)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[0].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[0].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[0].Node.Name)

	failures = InterceptGomegaFailures(func() {
		logrus.Infof("Waiting for connection recovery...")
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}

func TestNSMHealLocalDieNSMDOneNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testNSMHealLocalDieNSMDOneNode(t, nsmd_test_utils.DeployNSC, nsmd_test_utils.DeployICMP, nsmd_test_utils.CheckNSC)
}

func TestNSMHealLocalDieNSMDOneNodeMemif(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testNSMHealLocalDieNSMDOneNode(t, nsmd_test_utils.DeployVppAgentNSC, nsmd_test_utils.DeployVppAgentICMP, nsmd_test_utils.CheckVppAgentNSC)
}

func testNSMHealLocalDieNSMDOneNode(t *testing.T, deployNsc, deployNse nsmd_test_utils.PodSupplier, nscCheck nsmd_test_utils.NscChecker) {
	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup := nsmd_test_utils.SetupNodes(k8s, 1, defaultTimeout)

	// Run ICMP on latest node
	icmpPod := deployNse(k8s, nodes_setup[0].Node, "icmp-responder-nse-1", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())

	nscPodNode := deployNsc(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *nsmd_test_utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = nscCheck(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	logrus.Infof("Delete Local NSMD")
	k8s.DeletePods(nodes_setup[0].Nsmd)

	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("%s-recovered", nodes_setup[0].Nsmd.Name)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[0].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[0].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[0].Node.Name)

	failures = InterceptGomegaFailures(func() {
		logrus.Infof("Waiting for connection recovery...")
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = nscCheck(k8s, t, nscPodNode)
	})
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}
