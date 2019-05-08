// +build basic

package nsmd_integration_tests

import (
	"k8s.io/api/core/v1"
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSMgrRestartDeploy(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMgr Deploy test")

	k8s, err := kube_testing.NewK8s(true)
	//defer k8s.Cleanup()

	Expect(err).To(BeNil())

	nodes := k8s.GetNodesWait(2, defaultTimeout)

	if len(nodes) < 2 {
		logrus.Printf("At least two Kubernetes nodes are required for this test")
		Expect(len(nodes)).To(Equal(2))
		return
	}

	pods := nsmd_test_utils.SetupNodes(k8s, 1, defaultTimeout)

	result, result_err, err := k8s.Exec(pods[0].Nsmd, "nsmd", "kill", "-6", "1")
	logrus.Infof("Kill status %v %v %v", result, result_err, err)

	st := time.Now()
	restarts := int32(0)
	for {
		pod, err := k8s.GetPod(pods[0].Nsmd)
		if err != nil {
			logrus.Printf("error during recieve pod information %v %v", err, pod)
		}
		restarts = 0
		alive := 0
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				continue
			}
			restarts += cs.RestartCount
			alive += 1
		}
		if alive == len(pod.Status.ContainerStatuses) && restarts > 0 {
			// All are alive and we have restart
			break
		}
		if time.Since(st) > fastTimeout {
			logrus.Errorf("Failed to have NSmgr restarted")
			t.Fail()
		}
		<- time.After(100 * time.Millisecond)
	}
	k8s.WaitLogsContains(pods[0].Nsmd, "nsmd", "NSM gRPC API Server: [::]:5001 is operational", defaultTimeout)

	failures := InterceptGomegaFailures(func() {
		Expect(restarts).To(Equal(int32(1)))
		prevLogs, err := k8s.GetLogsWithOptions(pods[0].Nsmd, &v1.PodLogOptions{
			Container: "nsmd",
			Previous:  true,
		})
		Expect(err).To(BeNil())
		//logrus.Infof("Previous logs: %v", prevLogs)
		Expect(strings.Contains(prevLogs, "SIGABRT: abort")).To(Equal(true))

	})

	nsmd_test_utils.PrintErrors(failures, k8s, pods, nil, t)
}
