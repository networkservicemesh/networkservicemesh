// +build basic_suite

package integration

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/prefixcollector"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

func TestExcludePrefixCheck(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	utils.EnvVar(prefixcollector.ExcludedPrefixesEnv).Set("172.16.1.0/24")
	if kubetest.UseIPv6() {
		utils.EnvVar(prefixcollector.ExcludedPrefixesEnv).Set("100::/64")
	}

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup(t)
	g.Expect(err).To(BeNil())
	nodes, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.SaveTestArtifacts(t)

	icmp := kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-1", defaultTimeout)
	clientset, err := k8s.GetClientSet()
	g.Expect(err).To(BeNil())
	_, err = clientset.CoreV1().Pods(k8s.GetK8sNamespace()).Create(pods.NSCPod("nsc", nodes[0].Node,
		map[string]string{
			"CLIENT_LABELS":          "app=icmp",
			"CLIENT_NETWORK_SERVICE": "icmp-responder",
		},
	))

	k8s.WaitLogsContains(icmp, "", "IPAM: The available address pool is empty, probably intersected by excludedPrefix", defaultTimeout)

}
