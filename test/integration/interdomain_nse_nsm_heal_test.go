// +build interdomain

package nsmd_integration_tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestInterdomainNSMHealLocalDieNSMD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSMHeal(t, 2, 0, false)
}

func TestInterdomainNSMHealRemoteDieNSMD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSMHeal(t, 2, 1, false)
}

func TestInterdomainNSMHealRemoteDieNSMD_NSE(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSMHeal(t, 2, 1, true)
}

func testInterdomainNSMHeal(t *testing.T, clustersCount int, killIndex int, deleteNSE bool) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
		g.Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(g, true, kubeconfig)

		g.Expect(err).To(BeNil())

		config := []*pods.NSMgrPodConfig{}

		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.DataplaneVariables = kubetest.DefaultDataplaneVariables(k8s.GetForwardingPlane())

		config = append(config, cfg)

		nodesSetup, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, config, k8s.GetK8sNamespace())
		g.Expect(err).To(BeNil())

		k8ss = append(k8ss, &kubetest.ExtK8s{
			K8s:        k8s,
			NodesSetup: nodesSetup,
		})

		pnsmdName := fmt.Sprintf("pnsmgr-%s", nodesSetup[0].Node.Name)
		kubetest.DeployProxyNSMgr(k8s, nodesSetup[0].Node, pnsmdName, defaultTimeout)

		serviceCleanup := kubetest.RunProxyNSMgrService(k8s)
		defer serviceCleanup()

		defer k8ss[i].K8s.Cleanup()
	}

	// Run ICMP
	icmpPod := kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nseExternalIP, err := kubetest.GetNodeExternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
	if err != nil {
		nseExternalIP, err = kubetest.GetNodeInternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
		g.Expect(err).To(BeNil())
	}

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   fmt.Sprintf("icmp-responder@%s", nseExternalIP),
	})

	var nscInfo *kubetest.NSCCheckInfo

	failures := InterceptGomegaFailures(func() {
		nscInfo = kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)
		for i := 0; i < clustersCount; i++ {
			kubetest.PrintErrors(failures, k8ss[i].K8s, k8ss[i].NodesSetup, nscInfo, t)
			kubetest.ShowLogs(k8ss[i].K8s, t)
		}
		nscInfo.PrintLogs()

		t.Fail()
	}

	if killIndex == 0 {
		testInterdomainHealLocalNSMD(k8ss, clustersCount)
	} else {
		testInterdomainHealRemoteNSMD(k8ss, clustersCount, icmpPod, deleteNSE)
	}

	failures = InterceptGomegaFailures(func() {
		logrus.Infof("Waiting for connection recovery...")
		k8ss[0].K8s.WaitLogsContains(k8ss[0].NodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = kubetest.HealTestingPodFixture().CheckNsc(k8ss[0].K8s, nscPodNode)
	})

	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)
		for i := 0; i < clustersCount; i++ {
			kubetest.ShowLogs(k8ss[i].K8s, t)
		}
		nscInfo.PrintLogs()

		t.Fail()
	}
}

func testInterdomainHealLocalNSMD(k8ss []*kubetest.ExtK8s, clustersCount int) {

	logrus.Infof("Delete Local NSMD")
	k8ss[0].K8s.DeletePods(k8ss[0].NodesSetup[0].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8ss[clustersCount-1].K8s.WaitLogsContains(k8ss[clustersCount-1].NodesSetup[0].Nsmd, "nsmd", "NSM: Remote opened connection is not monitored and put into Healing state", defaultTimeout)

	// Now we are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("%s-recovered", k8ss[0].NodesSetup[0].Nsmd.Name)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	k8ss[0].NodesSetup[0].Nsmd = k8ss[0].K8s.CreatePod(
		pods.NSMgrPodWithConfig(
			nsmdName,
			k8ss[0].NodesSetup[0].Node,
			&pods.NSMgrPodConfig{Namespace: k8ss[0].K8s.GetK8sNamespace()},
		)) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), k8ss[0].NodesSetup[0].Node.Name)
}

func testInterdomainHealRemoteNSMD(k8ss []*kubetest.ExtK8s, clustersCount int, icmpPod *v1.Pod, deleteNSE bool) {

	logrus.Infof("Delete Remote NSMD")
	k8ss[clustersCount-1].K8s.DeletePods(k8ss[clustersCount-1].NodesSetup[0].Nsmd)

	if deleteNSE {
		logrus.Infof("Delete Remote ICMP responder NSE")
		k8ss[clustersCount-1].K8s.DeletePods(icmpPod)
	}

	logrus.Infof("Waiting for NSE with network service")
	k8ss[0].K8s.WaitLogsContains(k8ss[0].NodesSetup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	k8ss[clustersCount-1].NodesSetup[0].Nsmd = k8ss[clustersCount-1].K8s.CreatePod(
		pods.NSMgrPodWithConfig(
			nsmdName,
			k8ss[clustersCount-1].NodesSetup[0].Node,
			&pods.NSMgrPodConfig{Namespace: k8ss[clustersCount-1].K8s.GetK8sNamespace()},
		)) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), k8ss[0].NodesSetup[0].Node.Name)

	if deleteNSE {
		// Restore ICMP responder pod.
		kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[0].Node, "icmp-responder-nse-2", defaultTimeout)
	}
}
