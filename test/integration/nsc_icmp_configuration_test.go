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

const defaultTimeout = 60 * time.Second

func TestNSCAndICMPLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t,1)
}

func TestNSCAndICMPRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t,2)
}


/**
	If passed 1 both will be on same node, if not on different.
 */
func testNSCAndICMP(t *testing.T, nodesCount int ) {
	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "nsmd-dataplane", "icmp-responder-nse")
	logrus.Printf("Cleanup done: %v", time.Since(s1))
	nodes := k8s.GetNodesWait(nodesCount, time.Second*60)
	if len(nodes) < nodesCount {
		logrus.Printf("At least one kubernetes node are required for this test")
		Expect(len(nodes)).To(Equal(nodesCount))
		return
	}
	nsmdPodNode := []*v1.Pod{}
	nsmdDataplanePodNode := []*v1.Pod{}

	s1 = time.Now()
	for k := 0; k< nodesCount; k++ {
		corePodName := fmt.Sprintf("nsmd%d", k)
		dataPlanePodName := fmt.Sprintf("nsmd-dataplane%d",k)
		corePods := k8s.CreatePods(pods.NSMDPod(corePodName, &nodes[k]), pods.VPPDataplanePod(dataPlanePodName, &nodes[k]))
		logrus.Printf("Started NSMD/Dataplane: %v on node %d", time.Since(s1), k)
		nsmdPodNode = append(nsmdPodNode, corePods[0])
		nsmdDataplanePodNode = append(nsmdDataplanePodNode, corePods[1])

		Expect(corePods[0].Name).To(Equal(corePodName))
		Expect(corePods[1].Name).To(Equal(dataPlanePodName))

		k8s.WaitLogsContains(nsmdDataplanePodNode[k], "", "Sending MonitorMechanisms update", defaultTimeout)
		k8s.WaitLogsContains(nsmdPodNode[k], "nsmd", "Dataplane added", defaultTimeout)
		k8s.WaitLogsContains(nsmdPodNode[k], "nsmdp", "ListAndWatch was called with", defaultTimeout)
	}

	s1 = time.Now()
	logrus.Infof("Starting ICMP Responder NSE on node: %d", nodesCount-1)
	icmpPodNode := k8s.CreatePod(pods.ICMPResponderPod("icmp-responder-nse1", &nodes[nodesCount-1],
		map[string]string{
			"NSE_LABELS": "app=icmp-responder", "IP_ADDRESS": "10.20.1.0/24",
		},
	))
	Expect(icmpPodNode.Name).To(Equal("icmp-responder-nse1"))

	k8s.WaitLogsContains(icmpPodNode, "", "nse: channel has been successfully advertised, waiting for connection from NSM...", defaultTimeout)

	logrus.Printf("ICMP Responder started done: %v", time.Since(s1))

	s1 = time.Now()
	nscPodNode := k8s.CreatePod(pods.NSCPod("nsc1", &nodes[0],
		map[string]string{
			"NSC_LABELS": "app=icmp", "NETWORK_SERVICES": "nsm1:icmp-responder",
		},
	))
	Expect(nscPodNode.Name).To(Equal("nsc1"))

	k8s.WaitLogsContains(nscPodNode, "nsc", "nsm client: initialization is completed successfully", defaultTimeout)
	logrus.Printf("NSC started done: %v", time.Since(s1))

	var ipResponse string = ""
	var routeResponse string = ""
	var pingResponse string = ""
	var errOut string = ""
	failures := InterceptGomegaFailures(func() {
		ipResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr")
		Expect(err).To(BeNil())
		Expect(errOut).To(Equal(""))
		logrus.Printf("NSC IP status Ok")

		Expect(strings.Contains(ipResponse, "10.20.1.1")).To(Equal(true))
		Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

		routeResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "route")
		Expect(err).To(BeNil())
		Expect(errOut).To(Equal(""))
		logrus.Printf("NSC Route status, Ok")

		Expect(strings.Contains(routeResponse, "8.8.8.8")).To(Equal(true))
		Expect(strings.Contains(routeResponse, "nsm")).To(Equal(true))

		pingResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ping", "10.20.1.2", "-c", "5")
		Expect(err).To(BeNil())
		Expect(strings.Contains(pingResponse, "5 packets transmitted, 5 packets received, 0% packet loss")).To(Equal(true))
		logrus.Printf("NSC Ping is success:%s", pingResponse)
	})
	// Do dumping of container state to dig into what is happened.
	if len(failures) > 0 {
		logrus.Errorf("Failues: %v", failures)

		for k := 0; k < nodesCount; k++ {
			nsmdLogs, _ := k8s.GetLogs(nsmdPodNode[k], "nsmd")
			logrus.Errorf("===================== NSMD %d output since test is failing %v\n=====================", k, nsmdLogs)

			nsmdk8sLogs, _ := k8s.GetLogs(nsmdPodNode[k], "nsmd-k8s")
			logrus.Errorf("===================== NSMD K8S %d output since test is failing %v\n=====================", k, nsmdk8sLogs)

			nsmdpLogs, _ := k8s.GetLogs(nsmdPodNode[k], "nsmdp")
			logrus.Errorf("===================== NSMD K8S %d output since test is failing %v\n=====================", k, nsmdpLogs)

			dataplaneLogs, _ := k8s.GetLogs(nsmdDataplanePodNode[k], "")
			logrus.Errorf("===================== Dataplane %d output since test is failing %v\n=====================", k, dataplaneLogs)
		}

		logrus.Errorf("===================== NSC IP Addr %v\n=====================", ipResponse)
		logrus.Errorf("===================== NSC IP Route %v\n=====================", routeResponse)
		logrus.Errorf("===================== NSC IP PING %v\n=====================", pingResponse)

		t.Fail()
	}
}
