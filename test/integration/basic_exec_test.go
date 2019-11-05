// +build basic

package nsmd_integration_tests

import (
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestExec(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8sWithoutRoles(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)

	k8s.Prepare("alpine-pod")

	alpinePod := k8s.CreatePod(pods.AlpinePod("alpine-pod", nil))

	ipResponse, errResponse, error := k8s.Exec(alpinePod, alpinePod.Spec.Containers[0].Name, "ip", "addr")
	g.Expect(error).To(BeNil())
	g.Expect(errResponse).To(Equal(""))
	logrus.Printf("NSC IP status:%s", ipResponse)
	logrus.Printf("End of test")
	k8s.DeletePods(alpinePod)
}

func TestExecCheckReDeploy(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8sWithoutRoles(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)

	attempt := 0

	k8s.SetDescribeHook(func(pod *v1.Pod, events []v1.Event) []v1.Event {
		attempt += 1
		if attempt == 1 {
			logrus.Infof("return error msg")
			return append(events, v1.Event{
				InvolvedObject: v1.ObjectReference{
					Name: pod.Name,
					Kind: "pod",
				},
				Reason:  "",
				Message: fmt.Sprintf("Some Error %s and some more error", kubetest.NetworkPluginCNIFailure),
			})
		}
		logrus.Infof("normal deploy")
		return events
	})

	k8s.Prepare("alpine-pod")

	keeper := utils.NewLogKeeper()
	defer keeper.Stop()

	alpinePod := k8s.CreatePod(pods.AlpinePod("alpine-pod", nil))

	logrus.Infof("Messages: %v", keeper.GetMessages())

	ipResponse, errResponse, error := k8s.Exec(alpinePod, alpinePod.Spec.Containers[0].Name, "ip", "addr")
	g.Expect(error).To(BeNil())
	g.Expect(errResponse).To(Equal(""))
	logrus.Printf("NSC IP status:%s", ipResponse)
	logrus.Printf("End of test")

	g.Expect(keeper.CheckMessagesOrder([]string{
		"Creating pod alpine-pod attempt:1",
		"Attempt to re-deploy pod alpine-pod. Reason: <nil>, Delete it",
		"Deleting alpine-pod",
		"The POD \"alpine-pod\" Deleted",
		"Creating pod alpine-pod attempt:2",
		"normal deploy",
		"deployed: alpine-pod",
	})).To(Equal(true))

	k8s.DeletePods(alpinePod)
}
