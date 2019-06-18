// +build interdomain

package nsmd_integration_tests

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	"os"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestInterdomainNSCAndICMPRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSCAndICMP(t, 2, 1,false)
}

func TestInterdomainNSCAndICMPRemoteVeth(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSCAndICMP(t, 2, 1, true)
}

func TestInterdomainNSCAndICMPProxyRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainNSCAndICMP(t, 2, 2,false)
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testInterdomainNSCAndICMP(t *testing.T, clustersCount int, nodesCount int, disableVHost bool) {
	k8ss := []* kubetest.ExtK8s{}

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
		cfg.DataplaneVariables = kubetest.DefaultDataplaneVariables(k8s.GetForwardingPlane())
		if disableVHost {
			cfg.DataplaneVariables["DATAPLANE_ALLOW_VHOST"] = "false"
		}
		config = append(config, cfg)

		nodesSetup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
		Expect(err).To(BeNil())

		k8ss = append(k8ss, &kubetest.ExtK8s{
			K8s:      k8s,
			NodesSetup: nodesSetup,
		})
		defer k8ss[i].K8s.Cleanup()
	}

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8ss[clustersCount - 1].K8s, k8ss[clustersCount - 1].NodesSetup[nodesCount - 1].Node, "icmp-responder-nse-1", defaultTimeout)

	nseInternalIP, err := kubetest.GetNodeInternalIP(k8ss[clustersCount - 1].NodesSetup[0].Node)
	Expect(err).To(BeNil())

	nscPodNode := kubetest.DeployNSCWithEnv(k8ss[0].K8s, k8ss[0].NodesSetup[0].Node, "nsc-1", defaultTimeout, map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   fmt.Sprintf("icmp-responder@%s", nseInternalIP),
	})

	var nscInfo *kubetest.NSCCheckInfo

	failures := InterceptGomegaFailures(func() {
		nscInfo = kubetest.CheckNSC(k8ss[0].K8s, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)
		for i := 0; i < clustersCount; i++ {
			kubetest.PrintLogs(k8ss[i].K8s, k8ss[i].NodesSetup)
		}
		nscInfo.PrintLogs()

		t.Fail()
	}
}