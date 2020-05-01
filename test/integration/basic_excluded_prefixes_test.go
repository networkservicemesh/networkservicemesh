// +build basic

package integration

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/prefixcollector"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
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

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesCount := 1

	variables := map[string]string{
		nsmd.NsmdDeleteLocalRegistry: "true",
	}

	if k8s.UseIPv6() {
		variables = map[string]string{
			nsmd.NsmdDeleteLocalRegistry: "true",
		}
	}

	nodes, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables:          variables,
			ForwarderVariables: kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane()),
			Namespace:          k8s.GetK8sNamespace(),
		},
	}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	defer k8s.SaveTestArtifacts(t)

	icmp := kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-1", defaultTimeout)

	clientset, err := k8s.GetClientSet()
	g.Expect(err).To(BeNil())

	nsc, err := clientset.CoreV1().Pods(k8s.GetK8sNamespace()).Create(context.TODO(), pods.NSCPod("nsc", nodes[0].Node,
		map[string]string{
			"CLIENT_LABELS":          "app=icmp",
			"CLIENT_NETWORK_SERVICE": "icmp-responder",
		},
	), metaV1.CreateOptions{})

	defer k8s.DeletePods(nsc)

	g.Expect(err).To(BeNil())
	k8s.WaitLogsContains(icmp, "", "IPAM: The available address pool is empty, probably intersected by excludedPrefix", defaultTimeout)

}
