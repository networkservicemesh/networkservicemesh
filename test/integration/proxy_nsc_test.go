package nsmd_integration_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

func TestProxyNscLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testProxyNsc(t, 1)
}

func TestProxyNscRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testProxyNsc(t, 2)
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testProxyNsc(t *testing.T, nodesCount int) {
	RegisterTestingT(t)

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "proxy-nsc", "nsmd-dataplane", "vpn-gateway-nse")
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
	for k := 0; k < nodesCount; k++ {
		corePodName := fmt.Sprintf("nsmd%d", k)
		dataPlanePodName := fmt.Sprintf("nsmd-dataplane%d", k)
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
	logrus.Infof("Starting VPN Gateway NSE on node: %d", nodesCount-1)
	vpnGatewayPodNode := k8s.CreatePod(pods.VPNGatewayPod("vpn-gateway-nse1", &nodes[nodesCount-1],
		map[string]string{
			"ADVERTISE_NSE_NAME":   "secure-intranet-connectivity",
			"ADVERTISE_NSE_LABELS": "app=vpn-gateway",
			"IP_ADDRESS":           "10.60.1.0/24",
		},
	))
	Expect(vpnGatewayPodNode.Name).To(Equal("vpn-gateway-nse1"))

	k8s.WaitLogsContains(vpnGatewayPodNode, "vpn-gateway", "NSE: channel has been successfully advertised, waiting for connection from NSM...", time.Second)

	logrus.Printf("VPN Gateway started done: %v", time.Since(s1))

	s1 = time.Now()
	nscPodNode := k8s.CreatePod(pods.ProxyNSCPod("proxy-nsc1", &nodes[0],
		map[string]string{
			"PROXY_HOST":        ":8080",
			"OUTGOING_NSC_NAME": "secure-intranet-connectivity",
		},
	))
	Expect(nscPodNode.Name).To(Equal("proxy-nsc1"))

	k8s.WaitLogsContains(nscPodNode, "proxy-nsc", "proxy nsm client: initialization is completed successfully", defaultTimeout)
	logrus.Printf("Proxy NSC started done: %v", time.Since(s1))

	var wgetResponse string
	var failures []string
	for i := 1; i < 10; i++ {
		failures = InterceptGomegaFailures(func() {
			_, wgetResponse, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "wget", "-O", "/dev/null", "--timeout", "5", "http://localhost:8080")
			Expect(err).To(BeNil())
			Expect(strings.Contains(wgetResponse, "100% |*******************************|   112")).To(Equal(true))
			logrus.Printf("Proxy NSC wget request is succeeded: %s", wgetResponse)
		})
	}

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

		logrus.Errorf("===================== Proxy NSC WGET %v\n=====================", wgetResponse)

		t.Fail()
	}
}
