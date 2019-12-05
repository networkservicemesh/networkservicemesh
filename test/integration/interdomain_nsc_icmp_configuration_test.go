// +build interdomain

package nsmd_integration_tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestInterdomainNSCAndICMPRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSCAndICMP(t, 2, 1, false)
}

func TestInterdomainNSCAndICMPRemoteVeth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSCAndICMP(t, 2, 1, true)
}

func TestInterdomainNSCAndICMPProxyRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSCAndICMP(t, 2, 2, false)
}

func testInterdomainNSCAndICMP(t *testing.T, clustersCount int, nodesCount int, disableVHost bool) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
		g.Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(g, true, kubeconfig)
		g.Expect(err).To(BeNil())
		defer k8s.Cleanup()
		defer kubetest.MakeLogsSnapshot(k8s, t)

		config := []*pods.NSMgrPodConfig{}

		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.ForwarderVariables = kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane())
		if disableVHost {
			cfg.ForwarderVariables["FORWARDER_ALLOW_VHOST"] = "false"
		}
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
		"OUTGOING_NSC_LABELS":    "app=icmp",
		"CLIENT_NETWORK_SERVICE": fmt.Sprintf("icmp-responder@%s", nseExternalIP),
	})

	kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)
}
