package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

func TestNSMDDdataplabeDeploy(t *testing.T) {
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
	nodes := k8s.GetNodesWait(2, time.Second*60)

	if len(nodes) < 2 {
		logrus.Printf("At least two kubernetes nodes are required for this test")
		Expect(len(nodes)).To(Equal(2))
		return
	}

	nsmdPodNode1 := k8s.CreatePods(pods.NSMDPod("nsmd1", &nodes[0]))
	nsmdPodNode2 := k8s.CreatePods(pods.NSMDPod("nsmd2", &nodes[1]))

	for _, lpod := range k8s.ListPods() {
		logrus.Printf("Found pod %s %+v", lpod.Name, lpod.Status)
		if strings.Contains(lpod.Name, "nsmd") {
			Expect(lpod.Status.Phase).To(Equal(v1.PodRunning))
		}
	}

	k8s.DeletePods(nsmdPodNode1...)
	k8s.DeletePods(nsmdPodNode2...)
	var count int = 0
	for _, lpod := range k8s.ListPods() {
		logrus.Printf("Found pod %s %+v", lpod.Name, lpod.Status)
		if strings.Contains(lpod.Name, "nsmd") {
			count += 1
		}
	}
	Expect(count).To(Equal(int(0)))
}
