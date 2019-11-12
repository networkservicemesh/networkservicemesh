// +build basic

package nsmd_integration_tests

import (
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSMgrForwarderDeploy(t *testing.T) {
	testNSMgrForwarderDeploy(t, pods.NSMgrPod, pods.ForwardingPlane)
}

func TestNSMgrForwarderDeployLiveCheck(t *testing.T) {
	testNSMgrForwarderDeploy(t, pods.NSMgrPodLiveCheck, pods.ForwardingPlaneWithLiveCheck)
}

func testNSMgrForwarderDeploy(t *testing.T, nsmdPodFactory func(string, *v1.Node, string) *v1.Pod, forwarderPodFactory func(string, *v1.Node, string) *v1.Pod) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMgr Deploy test")

	k8s, err := kubetest.NewK8s(g, kubetest.DefaultClear)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)

	nodes := k8s.GetNodesWait(1, defaultTimeout)

	nsmdName := fmt.Sprintf("nsmgr-%s", nodes[0].Name)
	forwarderName := fmt.Sprintf("nsmd-forwarder-%s", nodes[0].Name)
	corePod := nsmdPodFactory(nsmdName, &nodes[0], k8s.GetK8sNamespace())
	forwarderPod := forwarderPodFactory(forwarderName, &nodes[0], k8s.GetForwardingPlane())
	corePods, err := k8s.CreatePodsRaw(defaultTimeout, true, corePod, forwarderPod)
	g.Expect(err).To(BeNil())

	k8s.WaitLogsContains(corePods[1], "", "Sending MonitorMechanisms update", defaultTimeout)
	_ = k8s.WaitLogsContainsRegex(corePods[0], "nsmd", "NSM gRPC API Server: .* is operational", defaultTimeout)
	k8s.WaitLogsContains(corePods[0], "nsmdp", "nsmdp: successfully started", defaultTimeout)
	k8s.WaitLogsContains(corePods[0], "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", defaultTimeout)

	k8s.Cleanup()
	var count int = 0
	for _, lpod := range k8s.ListPods() {
		logrus.Printf("Found pod %s %+v", lpod.Name, lpod.Status)
		if strings.Contains(lpod.Name, "nsmgr") {
			count += 1
		}
	}
	g.Expect(count).To(Equal(int(0)))
}
