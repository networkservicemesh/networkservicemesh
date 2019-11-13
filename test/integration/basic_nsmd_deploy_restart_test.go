// +build suite basic

package nsmd_integration_tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestNSMgrRestartDeploy(t *testing.T) {
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

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())

	prevLogsChan, errChan := readLogsFully(k8s, nodesConf[0].Nsmd, "nsmd")

	result, resultErr, err := k8s.Exec(nodesConf[0].Nsmd, "nsmd", "kill", "-6", "1")
	logrus.Infof("Kill status %v %v %v", result, resultErr, err)

	var restarts int32
	for st := time.Now(); ; <-time.After(100 * time.Millisecond) {
		if time.Since(st) > fastTimeout {
			t.Fatal("Failed to have NSmgr restarted")
		}

		pod, err := k8s.GetPod(nodesConf[0].Nsmd)
		if err != nil {
			logrus.Infof("error during recieve pod information %v %v", err, pod)
			continue
		}

		restarts = 0
		alive := 0
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				break
			}
			restarts += cs.RestartCount
			alive += 1
		}

		if alive == len(pod.Status.ContainerStatuses) && restarts > 0 {
			// All are alive and we have restart
			break
		}
	}

	_ = k8s.WaitLogsContainsRegex(nodesConf[0].Nsmd, "nsmd", "NSM gRPC API Server: .* is operational", defaultTimeout)

	g.Expect(restarts).To(Equal(int32(1)))
	g.Expect(<-errChan).To(BeNil())
	g.Expect(<-prevLogsChan).To(ContainSubstring("SIGABRT: abort"))
}

func readLogsFully(k8s *kubetest.K8s, pod *v1.Pod, container string) (chan string, chan error) {
	options := &v1.PodLogOptions{
		Container: container,
		Follow:    true,
	}

	logsChan := make(chan string, 1)
	errChan := make(chan error, 1)
	go func() {
		defer close(logsChan)
		defer close(errChan)

		logs, err := k8s.GetLogsWithOptions(pod, options)
		logsChan <- logs
		errChan <- err
	}()

	return logsChan, errChan
}
