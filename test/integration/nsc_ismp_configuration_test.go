package nsmd_integration_tests

import (
	"github.com/ligato/networkservicemesh/test/kube_testing"
	"github.com/ligato/networkservicemesh/test/kube_testing/pods"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSCAndICMPLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "nsmd-dataplane", "icmp-responder-nse")
	logrus.Printf("Cleanup done: %v", time.Since(s1))
	nodes := k8s.GetNodes()
	if len(nodes) < 2 {
		logrus.Printf("At least two kubernetes nodes are required for this test")
		Expect(len(nodes)).To(Equal(2))
		return
	}
	s1 = time.Now()
	corePods := k8s.CreatePods(pods.NSMDPod("nsmd1", &nodes[0]), pods.VPPDataplanePod("nsmd-dataplane", &nodes[0]))
	logrus.Printf("Started NSMD/Dataplane: %v", time.Since(s1))
	nsmdPodNode := corePods[0]
	nsmdDataplanePodNode := corePods[1]

	Expect(nsmdPodNode.Name).To(Equal("nsmd1"))
	Expect(nsmdDataplanePodNode.Name).To(Equal("nsmd-dataplane"))

	k8s.WaitLogsContains(nsmdDataplanePodNode, "", "Sending MonitorMechanisms update", 10*time.Second)
	k8s.WaitLogsContains(nsmdPodNode, "nsmd", "Dataplane added", 10*time.Second)
	k8s.WaitLogsContains(nsmdPodNode, "nsmdp", "ListAndWatch was called with", 10*time.Second)

	s1 = time.Now()
	icmpPodNode := k8s.CreatePod(pods.ICMPResponderPod("icmp-responder-nse1", &nodes[0],
		map[string]string{
			"NSE_LABELS": "app=icmp-responder", "IP_ADDRESS": "10.20.1.1",
		},
	))
	Expect(icmpPodNode.Name).To(Equal("icmp-responder-nse1"))

	k8s.WaitLogsContains(icmpPodNode, "", "nse: channel has been successfully advertised, waiting for connection from NSM...", 10*time.Second)

	logrus.Printf("ICMP Responder started done: %v", time.Since(s1))

	s1 = time.Now()
	nscPodNode := k8s.CreatePod(pods.NSCPod("nsc1", &nodes[0],
		map[string]string{
			"NSC_LABELS": "app=icmp", "NETWORK_SERVICES": "nsm1:icmp-responder",
		},
	))
	Expect(nscPodNode.Name).To(Equal("nsc1"))

	k8s.WaitLogsContains(nscPodNode, "nsc", "nsm client: initialization is completed successfully", 10*time.Second)
	logrus.Printf("NSC started done: %v", time.Since(s1))

	ipResponse, errOut, error := k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr")
	Expect(error).To(BeNil())
	Expect(errOut).To(Equal(""))
	logrus.Printf("NSC IP status:%s", ipResponse)

	Expect(strings.Contains(ipResponse, "10.20.1.1")).To(Equal(true))
	Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

	routeResponse, errOut, error := k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "route")
	Expect(error).To(BeNil())
	Expect(errOut).To(Equal(""))
	logrus.Printf("NSC Route status:%s", routeResponse)

	Expect(strings.Contains(routeResponse, "8.8.8.8")).To(Equal(true))
	Expect(strings.Contains(routeResponse, "nsm")).To(Equal(true))
}
