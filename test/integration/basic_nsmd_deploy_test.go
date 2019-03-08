// +build basic

package nsmd_integration_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

func TestNSMDDdataplaneDeploy(t *testing.T) {
	testNSMDDdataplaneDeploy(t, pods.NSMDPod, pods.VPPDataplanePod)
}

func TestNSMDDdataplaneDeployLiveCheck(t *testing.T) {
	testNSMDDdataplaneDeploy(t, pods.NSMDPodLiveCheck, pods.VPPDataplanePodLiveCheck)
}

func testNSMDDdataplaneDeploy(t *testing.T, nsmdPodFactory func(string, *v1.Node) *v1.Pod, dataplanePodFactory func(string, *v1.Node) *v1.Pod) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMD Deploy test")

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.Prepare("nsmd")
	nodes := k8s.GetNodesWait(2, defaultTimeout)

	if len(nodes) < 2 {
		logrus.Printf("At least two Kubernetes nodes are required for this test")
		Expect(len(nodes)).To(Equal(2))
		return
	}

	var pods []*v1.Pod

	for i, node := range nodes {
		nsmdPodName := fmt.Sprintf("nsmd-%d", i+1)
		dataPlanePodName := fmt.Sprintf("nsmd-dataplane-%d", i+1)
		podsNode := k8s.CreatePods(nsmdPodFactory(nsmdPodName, &node), dataplanePodFactory(dataPlanePodName, &node))

		k8s.WaitLogsContains(podsNode[0], "nsmd", "NSM gRPC API Server: [::]:5001 is operational", defaultTimeout)
		k8s.WaitLogsContains(podsNode[0], "nsmdp", "ListAndWatch was called with", defaultTimeout)
		k8s.WaitLogsContains(podsNode[1], "", "Sending MonitorMechanisms update", defaultTimeout)
		pods = append(pods, podsNode...)
	}

	k8s.DeletePods(pods...)
	var count int = 0
	for _, lpod := range k8s.ListPods() {
		logrus.Printf("Found pod %s %+v", lpod.Name, lpod.Status)
		if strings.Contains(lpod.Name, "nsmd") {
			count += 1
		}
	}
	Expect(count).To(Equal(int(0)))
}
