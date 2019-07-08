// +build basic

package nsmd_integration_tests

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSMgrDataplaneDeploy(t *testing.T) {
	testNSMgrDataplaneDeploy(t, 2, pods.NSMgrPod, pods.ForwardingPlane)
}

func TestNSMgrDataplaneDeployLiveCheck(t *testing.T) {
	testNSMgrDataplaneDeploy(t, 2, pods.NSMgrPodLiveCheck, pods.ForwardingPlaneWithLiveCheck)
}

func testNSMgrDataplaneDeploy(t *testing.T, nodesCount int, nsmdPodFactory func(string, *v1.Node, string) *v1.Pod, dataplanePodFactory func(string, *v1.Node, string) *v1.Pod) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMgr Deploy test")

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)

	nodes := k8s.GetNodesWait(nodesCount, defaultTimeout)

	if len(nodes) < nodesCount {
		logrus.Printf("At least two Kubernetes nodes are required for this test")
		Expect(len(nodes)).To(Equal(nodesCount))
		return
	}

	var wg sync.WaitGroup
	for i := 0; i < nodesCount; i++ {
		wg.Add(1)
		node := i
		go func() {
			defer wg.Done()
			nsmdName := fmt.Sprintf("nsmgr-%s", nodes[node].Name)
			dataplaneName := fmt.Sprintf("nsmd-dataplane-%s", nodes[node].Name)
			corePod := nsmdPodFactory(nsmdName, &nodes[node], k8s.GetK8sNamespace())
			dataplanePod := dataplanePodFactory(dataplaneName, &nodes[node], k8s.GetForwardingPlane())
			corePods, err := k8s.CreatePodsRaw(defaultTimeout, true, corePod, dataplanePod)
			Expect(err).To(BeNil())

			k8s.WaitLogsContains(corePods[1], "", "Sending MonitorMechanisms update", defaultTimeout)
			_ = k8s.WaitLogsContainsRegex(corePods[0], "nsmd", "NSM gRPC API Server: .* is operational", defaultTimeout)
			k8s.WaitLogsContains(corePods[0], "nsmdp", "nsmdp: successfully started", defaultTimeout)
			k8s.WaitLogsContains(corePods[0], "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", defaultTimeout)
		}()
	}
	wg.Wait()

	k8s.Cleanup()
	var count int = 0
	for _, lpod := range k8s.ListPods() {
		logrus.Printf("Found pod %s %+v", lpod.Name, lpod.Status)
		if strings.Contains(lpod.Name, "nsmgr") {
			count += 1
		}
	}
	Expect(count).To(Equal(int(0)))
}
