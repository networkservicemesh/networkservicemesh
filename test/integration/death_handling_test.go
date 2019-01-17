package nsmd_integration_tests

import (
	"fmt"
	"github.com/ligato/networkservicemesh/test/kube_testing"
	"github.com/ligato/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"strings"
	"testing"
	"time"
)

func deployNsmdAndDataplane(k8s *kube_testing.K8s, node *v1.Node) (nsmd *v1.Pod, dataplane *v1.Pod) {
	startTime := time.Now()

	nsmdName := fmt.Sprintf("nsmd-%s", node.Name)
	dataplaneName := fmt.Sprintf("nsmd-dataplane-%s", node.Name)
	corePods := k8s.CreatePods(pods.NSMDPod(nsmdName, node), pods.VPPDataplanePod(dataplaneName, node))
	logrus.Printf("Started NSMD/Dataplane: %v on node %s", time.Since(startTime), node.Name)
	nsmd = corePods[0]
	dataplane = corePods[1]

	Expect(nsmd.Name).To(Equal(nsmdName))
	Expect(dataplane.Name).To(Equal(dataplaneName))

	k8s.WaitLogsContains(dataplane, "", "Sending MonitorMechanisms update", defaultTimeout)
	k8s.WaitLogsContains(nsmd, "nsmd", "Dataplane added", defaultTimeout)
	k8s.WaitLogsContains(nsmd, "nsmdp", "ListAndWatch was called with", defaultTimeout)

	return
}

func deployIcmp(k8s *kube_testing.K8s, node *v1.Node) (icmp *v1.Pod) {
	startTime := time.Now()

	logrus.Infof("Starting ICMP Responder NSE on node: %s", node.Name)
	icmp = k8s.CreatePod(pods.ICMPResponderPod("icmp-responder-nse1", node,
		map[string]string{
			"ADVERTISE_NSE_NAME":   "icmp-responder",
			"ADVERTISE_NSE_LABELS": "app=icmp-responder",
			"IP_ADDRESS":           "10.20.1.0/24",
		},
	))
	Expect(icmp.Name).To(Equal("icmp-responder-nse1"))

	k8s.WaitLogsContains(icmp, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", defaultTimeout)

	logrus.Printf("ICMP Responder started done: %v", time.Since(startTime))
	return
}

func deployNsc(k8s *kube_testing.K8s, node *v1.Node) (nsc *v1.Pod) {
	startTime := time.Now()
	nsc = k8s.CreatePod(pods.NSCPod("nsc1", node,
		map[string]string{
			"OUTGOING_NSC_LABELS": "app=icmp",
			"OUTGOING_NSC_NAME":   "icmp-responder",
		},
	))
	Expect(nsc.Name).To(Equal("nsc1"))

	k8s.WaitLogsContains(nsc, "nsc", "nsm client: initialization is completed successfully", defaultTimeout)
	logrus.Printf("NSC started done: %v", time.Since(startTime))
	return
}

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

	nodes := k8s.GetNodesWait(nodesCount, time.Second*60)
	Expect(len(nodes) >= nodesCount).To(Equal(true),
		"At least one kubernetes node are required for this test")

	for i := 0; i < nodesCount; i++ {
		deployNsmdAndDataplane(k8s, &nodes[i])
	}

	icmp := deployIcmp(k8s, &nodes[nodesCount-1])
	nsc := deployNsc(k8s, &nodes[0])

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

		k8s.DeletePods("default", podToKill)
		time.Sleep(5 * time.Second)

		ipResponse, errOut, err = k8s.Exec(podToCheck, podToCheck.Spec.Containers[0].Name, "ip", "addr")
		Expect(err).To(BeNil())
		Expect(errOut).To(Equal(""))
		Expect(strings.Contains(ipResponse, "nsm")).To(Equal(false))
	})

	if len(failures) > 0 {
		logrus.Errorf("Failues: %v", failures)
		t.Fail()
	}
}
