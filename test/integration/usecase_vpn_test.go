// +build usecase

package nsmd_integration_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	nsapiv1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/crds"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

func TestVPNLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testVPN(t, 1, map[string]int{
		"vppagent-firewall-nse-1": 0,
		"vpn-gateway-nse-1":       0,
		"vpn-gateway-nsc-1":       0,
	}, false)
}

func TestVPNFirewallRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testVPN(t, 2, map[string]int{
		"vppagent-firewall-nse-1": 1,
		"vpn-gateway-nse-1":       0,
		"vpn-gateway-nsc-1":       0,
	}, false)
}

func TestVPNNSERemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testVPN(t, 2, map[string]int{
		"vppagent-firewall-nse-1": 0,
		"vpn-gateway-nse-1":       1,
		"vpn-gateway-nsc-1":       0,
	}, false)
}

func TestVPNNSCRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testVPN(t, 2, map[string]int{
		"vppagent-firewall-nse-1": 0,
		"vpn-gateway-nse-1":       0,
		"vpn-gateway-nsc-1":       1,
	}, false)
}

func testVPN(t *testing.T, nodesCount int, affinity map[string]int, verbose bool) {
	RegisterTestingT(t)

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.PrepareDefault()
	logrus.Printf("Cleanup done: %v", time.Since(s1))
	nodes := k8s.GetNodesWait(nodesCount, defaultTimeout)
	if len(nodes) < nodesCount {
		logrus.Printf("At least one Kubernetes node is required for this test")
		Expect(len(nodes)).To(Equal(nodesCount))
		return
	}
	nsmdPodNode := []*v1.Pod{}
	nsmdDataplanePodNode := []*v1.Pod{}

	s1 = time.Now()
	for k := 0; k < nodesCount; k++ {
		corePodName := fmt.Sprintf("nsmgr-%d", k)
		dataPlanePodName := fmt.Sprintf("nsmd-dataplane-%d", k)
		corePods := k8s.CreatePods(pods.NSMgrPod(corePodName, &nodes[k], k8s.GetK8sNamespace()), pods.VPPDataplanePod(dataPlanePodName, &nodes[k]))
		logrus.Printf("Started NSMD/Dataplane: %v on node %d", time.Since(s1), k)
		nsmdPodNode = append(nsmdPodNode, corePods[0])
		nsmdDataplanePodNode = append(nsmdDataplanePodNode, corePods[1])

		Expect(corePods[0].Name).To(Equal(corePodName))
		Expect(corePods[1].Name).To(Equal(dataPlanePodName))

		k8s.WaitLogsContains(nsmdDataplanePodNode[k], "", "Sending MonitorMechanisms update", defaultTimeout)
		k8s.WaitLogsContains(nsmdPodNode[k], "nsmd", "Dataplane added", defaultTimeout)
		k8s.WaitLogsContains(nsmdPodNode[k], "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", defaultTimeout)
		k8s.WaitLogsContains(nsmdPodNode[k], "nsmdp", "ListAndWatch was called with", defaultTimeout)
	}

	{
		nscrd, err := crds.NewNSCRD(k8s.GetK8sNamespace())
		Expect(err).To(BeNil())

		nsSecureIntranetConnectivity := crds.SecureIntranetConnectivity()
		logrus.Printf("About to insert: %v", nsSecureIntranetConnectivity)
		var result *nsapiv1.NetworkService
		result, err = nscrd.Create(nsSecureIntranetConnectivity)
		Expect(err).To(BeNil())
		logrus.Printf("CRD applied with result: %v", result)
		result, err = nscrd.Get(nsSecureIntranetConnectivity.ObjectMeta.Name)
		Expect(err).To(BeNil())
		logrus.Printf("Registered CRD is: %v", result)
	}

	s1 = time.Now()
	node := affinity["vppagent-firewall-nse-1"]
	logrus.Infof("Starting VPPAgent Firewall NSE on node: %d", node)
	_, err = k8s.CreateConfigMap(pods.VppAgentFirewallNSEConfigMapIcmpHttp("vppagent-firewall-nse-1", k8s.GetK8sNamespace()))
	Expect(err).To(BeNil())
	vppagentFirewallNode := k8s.CreatePod(pods.VppAgentFirewallNSEPodWithConfigMap("vppagent-firewall-nse-1", &nodes[node],
		map[string]string{
			"ADVERTISE_NSE_NAME":   "secure-intranet-connectivity",
			"ADVERTISE_NSE_LABELS": "app=firewall",
			"OUTGOING_NSC_NAME":    "secure-intranet-connectivity",
			"OUTGOING_NSC_LABELS":  "app=firewall",
		},
	))
	Expect(vppagentFirewallNode.Name).To(Equal("vppagent-firewall-nse-1"))

	k8s.WaitLogsContains(vppagentFirewallNode, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", fastTimeout)

	logrus.Printf("VPN Gateway started done: %v", time.Since(s1))

	s1 = time.Now()
	node = affinity["vpn-gateway-nse-1"]
	logrus.Infof("Starting VPN Gateway NSE on node: %d", node)
	vpnGatewayPodNode := k8s.CreatePod(pods.VPNGatewayNSEPod("vpn-gateway-nse-1", &nodes[node],
		map[string]string{
			"ADVERTISE_NSE_NAME":   "secure-intranet-connectivity",
			"ADVERTISE_NSE_LABELS": "app=vpn-gateway",
			"IP_ADDRESS":           "10.60.1.0/24",
		},
	))
	Expect(vpnGatewayPodNode.Name).To(Equal("vpn-gateway-nse-1"))

	k8s.WaitLogsContains(vpnGatewayPodNode, "vpn-gateway", "NSE: channel has been successfully advertised, waiting for connection from NSM...", fastTimeout)

	logrus.Printf("VPN Gateway started done: %v", time.Since(s1))

	s1 = time.Now()
	node = affinity["vpn-gateway-nsc-1"]
	nscPodNode := k8s.CreatePod(pods.NSCPod("vpn-gateway-nsc-1", &nodes[node],
		map[string]string{
			"OUTGOING_NSC_NAME": "secure-intranet-connectivity",
		},
	))
	Expect(nscPodNode.Name).To(Equal("vpn-gateway-nsc-1"))

	k8s.WaitLogsContains(nscPodNode, "nsc", "nsm client: initialization is completed successfully", defaultTimeout)
	logrus.Printf("VPN Gateway NSC started done: %v", time.Since(s1))

	var ipResponse = ""
	var routeResponse = ""
	var pingResponse = ""
	var errOut = ""
	var wgetResponse string
	var failures []string

	failures = InterceptGomegaFailures(func() {
		ipResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr")
		Expect(err).To(BeNil())
		Expect(errOut).To(Equal(""))
		logrus.Printf("NSC IP status Ok")

		Expect(strings.Contains(ipResponse, "10.60.1.1")).To(Equal(true))
		Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

		routeResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "route")
		Expect(err).To(BeNil())
		Expect(errOut).To(Equal(""))
		logrus.Printf("NSC Route status, Ok")

		Expect(strings.Contains(routeResponse, "8.8.8.8")).To(Equal(true))
		Expect(strings.Contains(routeResponse, "nsm")).To(Equal(true))
		for i := 1; i <= 1; i++ {
			pingResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ping", "10.60.1.2", "-A", "-c", "10")
			Expect(err).To(BeNil())
			Expect(strings.Contains(pingResponse, "10 packets received")).To(Equal(true))
			logrus.Printf("VPN NSC Ping succeeded:%s", pingResponse)

			_, wgetResponse, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "wget", "-O", "/dev/null", "--timeout", "3", "http://10.60.1.2:80")
			Expect(err).To(BeNil())
			Expect(strings.Contains(wgetResponse, "100% |***")).To(Equal(true))
			logrus.Printf("%d VPN NSC wget request succeeded: %s", i, wgetResponse)

			_, wgetResponse, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "wget", "-O", "/dev/null", "--timeout", "3", "http://10.60.1.2:8080")
			Expect(err).To(Not(BeNil()))
			Expect(strings.Contains(wgetResponse, "download timed out")).To(Equal(true))
			logrus.Printf("%d VPN NSC wget request succeeded: %s", i, wgetResponse)
		}
	})

	// Do dumping of container state to dig into what is happened.
	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)

		if verbose {
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
		}
		logrus.Errorf("===================== VPN NSC WGET %v\n=====================", wgetResponse)

		t.Fail()
	}
}
