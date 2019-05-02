// +build basic

package nsmd_integration_tests

import (
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestExcludePrefixCheck(t *testing.T) {
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

	nodesCount := 1

	variables := map[string]string{
		nsmd.ExcludedPrefixesEnv:     "172.16.1.0/24",
		nsmd.NsmdDeleteLocalRegistry: "true",
	}
	nodes := nsmd_test_utils.SetupNodesConfig(k8s, nodesCount, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: variables,
			Namespace: k8s.GetK8sNamespace(),
		},
	}, k8s.GetK8sNamespace())

	defer nsmd_test_utils.FailLogger(k8s, nodes, t)

	icmp := nsmd_test_utils.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-1", defaultTimeout)

	clientset, err := k8s.GetClientSet()
	Expect(err).To(BeNil())

	nsc, err := clientset.CoreV1().Pods(k8s.GetK8sNamespace()).Create(pods.NSCPod("nsc", nodes[0].Node,
		map[string]string{
			"OUTGOING_NSC_LABELS": "app=icmp",
			"OUTGOING_NSC_NAME":   "icmp-responder",
		},
	))

	defer k8s.DeletePods(nsc)

	Expect(err).To(BeNil())
	k8s.WaitLogsContains(icmp, "", "IPAM: The available address pool is empty, probably intersected by excludedPrefix", defaultTimeout)

}
