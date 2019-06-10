// +build usecase, interdomain

package nsmd_integration_tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"k8s.io/api/core/v1"
)

type ExtK8s struct {
	k8s *kubetest.K8s
	nodesSetup []*kubetest.NodeConf
}

func TestInterdomainNSCAndICMPRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSCAndICMP(t, 2,false, nil)
}

func TestInterdomainNSCAndICMPRemoteVeth(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSCAndICMP(t, 2, true, nil)
}

type HealFunc func(*testing.T, []*ExtK8s, int, *v1.Pod)

/**
If passed 1 both will be on same node, if not on different.
*/
func testInterdomainNSCAndICMP(t *testing.T, clustersCount int, disableVHost bool, healFunc HealFunc) {
	k8ss := []* ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i + 1))
		Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(true, kubeconfig)

		Expect(err).To(BeNil())

		config := []*pods.NSMgrPodConfig{}

		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.DataplaneVariables = kubetest.DefaultDataplaneVariables()
		if disableVHost {
			cfg.DataplaneVariables["DATAPLANE_ALLOW_VHOST"] = "false"
		}
		config = append(config, cfg)

		nodesSetup, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, config, k8s.GetK8sNamespace())
		Expect(err).To(BeNil())

		k8ss = append(k8ss, &ExtK8s{
			k8s:      k8s,
			nodesSetup: nodesSetup,
		})
		defer k8ss[i].k8s.Cleanup()
	}

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8ss[0].k8s, k8ss[0].nodesSetup[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nseInternalIP, err := kubetest.GetNodeInternalIP(k8ss[0].nodesSetup[0].Node)
	Expect(err).To(BeNil())

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[1].k8s, k8ss[1].nodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   fmt.Sprintf("icmp-responder@%s", nseInternalIP),
	})

	var nscInfo *kubetest.NSCCheckInfo

	failures := InterceptGomegaFailures(func() {
		nscInfo = kubetest.CheckNSC(k8ss[1].k8s, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)
		for i := 0; i < clustersCount; i++ {
			kubetest.PrintLogs(k8ss[i].k8s, k8ss[i].nodesSetup)
		}
		nscInfo.PrintLogs()

		t.Fail()
	}

	if healFunc != nil {
		logrus.Printf("Has func")
	}
}

func testInterdomainDataplaneHeal(t *testing.T, k8ss []*ExtK8s, killIndex int, nscPod *v1.Pod) {
	logrus.Infof("Delete Selected dataplane")
	k8ss[killIndex].k8s.DeletePods(k8ss[killIndex].nodesSetup[0].Dataplane)

	logrus.Infof("Wait NSMD is waiting for dataplane recovery")
	k8ss[killIndex].k8s.WaitLogsContains(k8ss[killIndex].nodesSetup[0].Nsmd, "nsmd", "Waiting for Dataplane to recovery...", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	dpName := fmt.Sprintf("nsmd-dataplane-recovered-%d", killIndex)

	logrus.Infof("Starting recovered dataplane...")
	startTime := time.Now()
	k8ss[killIndex].nodesSetup[0].Dataplane = k8ss[killIndex].k8s.CreatePod(pods.ForwardingPlane(dpName, k8ss[killIndex].nodesSetup[0].Node, k8ss[killIndex].k8s.GetForwardingPlane()))
	logrus.Printf("Started new Dataplane: %v on node %s", time.Since(startTime), k8ss[killIndex].nodesSetup[0].Node.Name)

	// Check NSMd goint into HEAL state.

	logrus.Infof("Waiting for connection recovery...")
	if killIndex != 0 {
		k8ss[killIndex].k8s.WaitLogsContains(k8ss[killIndex].nodesSetup[0].Nsmd, "nsmd", "Healing will be continued on source side...", defaultTimeout)
		k8ss[0].k8s.WaitLogsContains(k8ss[0].nodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	} else {
		k8ss[killIndex].k8s.WaitLogsContains(k8ss[killIndex].nodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	}
	logrus.Infof("Waiting for connection recovery Done...")

	var nscInfo *kubetest.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = kubetest.CheckNSC(k8ss[1].k8s, nscPod)
	})
	kubetest.PrintErrors(failures, k8ss[1].k8s, k8ss[1].nodesSetup, nscInfo, t)
}