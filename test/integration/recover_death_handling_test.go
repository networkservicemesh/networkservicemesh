// +build recover

package nsmd_integration_tests

import (
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

func TestNscDiesSingleNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, true, 1)
}

func TestNseDiesSingleNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, false, 1)
}

func TestNscDiesMultiNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, true, 2)
}

func TestNseDiesMultiNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, false, 2)
}

func testDie(t *testing.T, killSrc bool, nodesCount int) {
	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "nsmd-dataplane", "icmp-responder-nse")
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	nodes := nsmd_test_utils.SetupNodes(k8s, nodesCount, defaultTimeout)

	icmp := nsmd_test_utils.DeployIcmp(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse1", defaultTimeout)
	nsc := nsmd_test_utils.DeployNsc(k8s, nodes[0].Node, "nsc1", defaultTimeout)

	failures := InterceptGomegaFailures(func() {
		ipResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ip", "addr")
		Expect(err).To(BeNil())
		Expect(errOut).To(Equal(""))
		Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

		ipResponse, errOut, err = k8s.Exec(icmp, icmp.Spec.Containers[0].Name, "ip", "addr")
		Expect(err).To(BeNil())
		Expect(errOut).To(Equal(""))
		Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

		pingResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ping", "10.20.1.2", "-c", "5")
		Expect(err).To(BeNil())
		Expect(strings.Contains(pingResponse, "5 packets transmitted, 5 packets received, 0% packet loss")).To(Equal(true))
		logrus.Printf("NSC Ping is success:%s", pingResponse)

		var podToKill *v1.Pod
		var podToCheck *v1.Pod
		if killSrc {
			podToKill = nsc
			podToCheck = icmp
		} else {
			podToKill = icmp
			podToCheck = nsc
		}

		k8s.DeletePods(podToKill)
		success := false
		for attempt := 0; attempt < 20; <-time.Tick(300 * time.Millisecond) {
			attempt++
			ipResponse, errOut, err = k8s.Exec(podToCheck, podToCheck.Spec.Containers[0].Name, "ip", "addr")
			if !strings.Contains(ipResponse, "nsm") {
				success = true
				break
			}
		}
		Expect(success).To(Equal(true))
	})

	if len(failures) > 0 {
		logrus.Errorf("Failues: %v", failures)
		for k := 0; k < nodesCount; k++ {
			nsmdLogs, _ := k8s.GetLogs(nodes[k].Nsmd, "nsmd")
			logrus.Errorf("===================== NSMD %d output since test is failing %v\n=====================", k, nsmdLogs)

			nsmdk8sLogs, _ := k8s.GetLogs(nodes[k].Nsmd, "nsmd-k8s")
			logrus.Errorf("===================== NSMD K8S %d output since test is failing %v\n=====================", k, nsmdk8sLogs)

			nsmdpLogs, _ := k8s.GetLogs(nodes[k].Nsmd, "nsmdp")
			logrus.Errorf("===================== NSMD K8S %d output since test is failing %v\n=====================", k, nsmdpLogs)

			dataplaneLogs, _ := k8s.GetLogs(nodes[k].Dataplane, "")
			logrus.Errorf("===================== Dataplane %d output since test is failing %v\n=====================", k, dataplaneLogs)
		}

		t.Fail()
	}
}
