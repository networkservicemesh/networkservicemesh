// +build basic

package nsmd_integration_tests

import (
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestProxyNSMgrDeployLiveCheck(t *testing.T) {
	testProxyNSMgrDeploy(t, pods.ProxyNSMgrPodLiveCheck)
}

func testProxyNSMgrDeploy(t *testing.T, proxyNsmdPodFactory func(string, *v1.Node, string) *v1.Pod) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMgr Deploy test")

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResouces)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)

	nodes := k8s.GetNodesWait(1, defaultTimeout)

	pnsmdTemplate := proxyNsmdPodFactory("pnsmgr", &nodes[0], k8s.GetK8sNamespace())
	_, err = k8s.CreatePodsRaw(defaultTimeout, true, pnsmdTemplate)
	g.Expect(err).To(BeNil())

	k8s.Cleanup()
	count := 0
	for _, lpod := range k8s.ListPods() {
		logrus.Printf("Found pod %s %+v", lpod.Name, lpod.Status)
		if strings.Contains(lpod.Name, "pnsmgr") {
			count += 1
		}
	}
	g.Expect(count).To(Equal(int(0)))
}
