// +build interdomain

package integration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestInterdomainNSCAndICMPForwarderHealLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainForwarderHeal(t, 2, 2, 0)
}

func TestInterdomainNSCAndICMPForwarderHealRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainForwarderHeal(t, 2, 2, 1)
}

func testInterdomainForwarderHeal(t *testing.T, clustersCount int, nodesCount int, killIndex int) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
		g.Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(g, true, kubeconfig)
		g.Expect(err).To(BeNil())
		defer k8s.Cleanup()
		defer k8s.SaveTestArtifacts(t)

		config := []*pods.NSMgrPodConfig{}

		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.ForwarderVariables = kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane())

		config = append(config, cfg)

		nodesSetup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
		g.Expect(err).To(BeNil())

		k8ss = append(k8ss, &kubetest.ExtK8s{
			K8s:        k8s,
			NodesSetup: nodesSetup,
		})

		for j := 0; j < nodesCount; j++ {
			pnsmdName := fmt.Sprintf("pnsmgr-%s", nodesSetup[j].Node.Name)
			kubetest.DeployProxyNSMgr(k8s, nodesSetup[j].Node, pnsmdName, defaultTimeout)
		}

		serviceCleanup := kubetest.RunProxyNSMgrService(k8s)
		defer serviceCleanup()
	}

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8ss[clustersCount-1].K8s, k8ss[clustersCount-1].NodesSetup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	nseExternalIP, err := kubetest.GetNodeExternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
	if err != nil {
		nseExternalIP, err = kubetest.GetNodeInternalIP(k8ss[clustersCount-1].NodesSetup[0].Node)
		g.Expect(err).To(BeNil())
	}

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"CLIENT_LABELS":          "app=icmp",
		"CLIENT_NETWORK_SERVICE": fmt.Sprintf("icmp-responder@%s", nseExternalIP),
	})

	kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)

	nodeKillIndex := 0
	if killIndex > 0 {
		nodeKillIndex = nodesCount - 1
	}

	logrus.Infof("Delete Selected forwarder")
	k8ss[killIndex].K8s.DeletePods(k8ss[killIndex].NodesSetup[nodeKillIndex].Forwarder)

	logrus.Infof("Wait NSMD is waiting for forwarder recovery")
	k8ss[killIndex].K8s.WaitLogsContains(k8ss[killIndex].NodesSetup[nodeKillIndex].Nsmd, "nsmd", "Waiting for Forwarder to recovery...", defaultTimeout)
	// Now are are in forwarder dead state, and in Heal procedure waiting for forwarder.
	dpName := fmt.Sprintf("nsmd-forwarder-recovered-%d", killIndex)

	logrus.Infof("Starting recovered forwarder...")
	startTime := time.Now()
	k8ss[killIndex].NodesSetup[0].Forwarder = k8ss[killIndex].K8s.CreatePod(pods.ForwardingPlane(dpName, k8ss[killIndex].NodesSetup[nodeKillIndex].Node, k8ss[killIndex].K8s.GetForwardingPlane()))
	logrus.Printf("Started new Forwarder: %v on node %s", time.Since(startTime), k8ss[killIndex].NodesSetup[nodeKillIndex].Node.Name)

	// Check NSMd goint into HEAL state.

	logrus.Infof("Waiting for connection recovery...")
	if killIndex != 0 {
		k8ss[killIndex].K8s.WaitLogsContains(k8ss[killIndex].NodesSetup[nodeKillIndex].Nsmd, "nsmd", "Healing will be continued on source side...", defaultTimeout)
		k8ss[0].K8s.WaitLogsContains(k8ss[0].NodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	} else {
		k8ss[killIndex].K8s.WaitLogsContains(k8ss[killIndex].NodesSetup[nodeKillIndex].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	}
	logrus.Infof("Waiting for connection recovery Done...")

	kubetest.DefaultTestingPodFixture(g).CheckNsc(k8ss[0].K8s, nscPodNode)
}
