// +build recover,!azure

package nsmd_integration_tests

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSMHealRemoteDieNSMD_NSE(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.PrepareDefault()
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	// Deploy open tracing to see what happening.
	nodes_setup := nsmd_test_utils.SetupNodesConfig(k8s, 2, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsm.NsmdHealDSTWaitTimeout: "60", // 60 second delay, since we know on CI it could not fit into delay.
			},
		}, {},
	})

	// Run ICMP on latest node
	icmpPod := nsmd_test_utils.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := nsmd_test_utils.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *nsmd_test_utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	logrus.Infof("Delete Remote NSMD")
	k8s.DeletePods(nodes_setup[1].Nsmd)
	k8s.DeletePods(icmpPod)
	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder. Since elapsed:", 60*time.Second)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)
	time.Sleep(10 * time.Second)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	failures = InterceptGomegaFailures(func() {
		// Restore ICMP responder pod.
		icmpPod = nsmd_test_utils.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-2", defaultTimeout)

		logrus.Infof("Waiting for connection recovery...")
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", 60*time.Second)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}

func TestNSMHealRemoteDieNSMD(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.PrepareDefault()
	logrus.Printf("Cleanup done: %v", time.Since(s1))

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

	logrus.Infof("Delete Remote NSMD")
	k8s.DeletePods(nodes_setup[1].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder. Since elapsed:", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	failures = InterceptGomegaFailures(func() {
		logrus.Infof("Waiting for connection recovery...")
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}
