// +build usecase_suite

package nsmd_integration_tests

import (
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nsapiv1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/crds"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

const (
	ptNum = 5 // Number of Passthrough Endpoints to deploy
)

func TestVPNLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testVPN(t, ptNum, 1, map[string]int{
		"vppagent-firewall-nse-1":  0,
		"vppagent-passthrough-nse": 0,
		"vpn-gateway-nse-1":        0,
		"vpn-gateway-nsc-1":        0,
	}, false)
}

func TestVPNFirewallRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testVPN(t, ptNum, 2, map[string]int{
		"vppagent-firewall-nse-1":  1,
		"vppagent-passthrough-nse": 0,
		"vpn-gateway-nse-1":        0,
		"vpn-gateway-nsc-1":        0,
	}, false)
}

func TestVPNNSERemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testVPN(t, ptNum, 2, map[string]int{
		"vppagent-firewall-nse-1":  0,
		"vppagent-passthrough-nse": 0,
		"vpn-gateway-nse-1":        1,
		"vpn-gateway-nsc-1":        0,
	}, false)
}

func TestVPNNSCRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testVPN(t, ptNum, 2, map[string]int{
		"vppagent-firewall-nse-1":  0,
		"vppagent-passthrough-nse": 0,
		"vpn-gateway-nse-1":        0,
		"vpn-gateway-nsc-1":        1,
	}, false)
}

func testVPN(t *testing.T, ptnum, nodesCount int, affinity map[string]int, verbose bool) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())

	if k8s.UseIPv6() && nodesCount == 1 && !kubetest.IsBrokeTestsEnabled() {
		t.Skip("IPv6 usecase is temporarily broken for single node setups.")
		return
	}

	nodes, err := kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	g.Expect(err).To(BeNil())
	s1 := time.Now()

	_, err = kubetest.SetupNodes(k8s, nodesCount, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)

	{
		nscrd, err := crds.NewNSCRD(k8s.GetK8sNamespace())
		g.Expect(err).To(BeNil())

		nsSecureIntranetConnectivity := crds.SecureIntranetConnectivity(ptnum)
		logrus.Printf("About to insert: %v", nsSecureIntranetConnectivity)
		var result *nsapiv1.NetworkService
		result, err = nscrd.Create(nsSecureIntranetConnectivity)
		g.Expect(err).To(BeNil())
		logrus.Printf("CRD applied with result: %v", result)
		result, err = nscrd.Get(nsSecureIntranetConnectivity.ObjectMeta.Name)
		g.Expect(err).To(BeNil())
		logrus.Printf("Registered CRD is: %v", result)
		defer nscrd.Delete(result.Name, &metaV1.DeleteOptions{})

	}

	pingCommand := "ping"
	addressPool := "172.16.1.0/24"
	srcIP, dstIP := "172.16.1.1", "172.16.1.2"

	/* Change stuff related to IPv6 */
	if k8s.UseIPv6() {
		pingCommand = "ping6"
		addressPool = "100::/64"
		srcIP, dstIP = "100::1", "100::2"
	}

	s1 = time.Now()
	node := affinity["vppagent-firewall-nse-1"]
	logrus.Infof("Starting VPPAgent Firewall NSE on node: %d", node)
	_, err = k8s.CreateConfigMap(pods.VppAgentFirewallNSEConfigMapICMPHTTP("vppagent-firewall-nse-1", k8s.GetK8sNamespace()))
	g.Expect(err).To(BeNil())
	vppagentFirewallNode := k8s.CreatePod(pods.VppAgentFirewallNSEPodWithConfigMap("vppagent-firewall-nse-1", nodes[node].Node,
		map[string]string{
			"ADVERTISE_NSE_NAME":   "secure-intranet-connectivity",
			"ADVERTISE_NSE_LABELS": "app=firewall",
			"OUTGOING_NSC_NAME":    "secure-intranet-connectivity",
			"OUTGOING_NSC_LABELS":  "app=firewall",
		},
	))
	g.Expect(vppagentFirewallNode.Name).To(Equal("vppagent-firewall-nse-1"))

	k8s.WaitLogsContains(vppagentFirewallNode, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", fastTimeout)

	logrus.Printf("VPN firewall started done: %v", time.Since(s1))

	for i := 1; i <= ptnum; i++ {
		s1 = time.Now()
		id := strconv.Itoa(i)
		node = affinity["vppagent-passthrough-nse"]
		logrus.Infof("Starting VPPAgent Passthrough NSE on node: %d", node)

		vppagentPassthroughNode := k8s.CreatePod(pods.VppAgentFirewallNSEPod("vppagent-passthrough-nse-"+id, nodes[node].Node,
			map[string]string{
				"ADVERTISE_NSE_NAME":   "secure-intranet-connectivity",
				"ADVERTISE_NSE_LABELS": "app=passthrough-" + id,
				"OUTGOING_NSC_NAME":    "secure-intranet-connectivity",
				"OUTGOING_NSC_LABELS":  "app=passthrough-" + id,
			},
		))
		g.Expect(vppagentPassthroughNode.Name).To(Equal("vppagent-passthrough-nse-" + id))

		k8s.WaitLogsContains(vppagentPassthroughNode, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", fastTimeout)

		logrus.Printf("VPN passthrough started done: %v", time.Since(s1))
	}

	s1 = time.Now()
	node = affinity["vpn-gateway-nse-1"]
	logrus.Infof("Starting VPN Gateway NSE on node: %d", node)
	vpnGatewayPodNode := k8s.CreatePod(pods.VPNGatewayNSEPod("vpn-gateway-nse-1", nodes[node].Node,
		map[string]string{
			"ADVERTISE_NSE_NAME":   "secure-intranet-connectivity",
			"ADVERTISE_NSE_LABELS": "app=vpn-gateway",
			"IP_ADDRESS":           addressPool,
		},
	))
	g.Expect(vpnGatewayPodNode).ToNot(BeNil())
	g.Expect(vpnGatewayPodNode.Name).To(Equal("vpn-gateway-nse-1"))

	k8s.WaitLogsContains(vpnGatewayPodNode, "vpn-gateway", "NSE: channel has been successfully advertised, waiting for connection from NSM...", fastTimeout)

	logrus.Printf("VPN Gateway started done: %v", time.Since(s1))

	s1 = time.Now()
	node = affinity["vpn-gateway-nsc-1"]
	nscPodNode := k8s.CreatePod(pods.NSCPod("vpn-gateway-nsc-1", nodes[node].Node,
		map[string]string{
			"OUTGOING_NSC_NAME": "secure-intranet-connectivity",
		},
	))
	g.Expect(nscPodNode.Name).To(Equal("vpn-gateway-nsc-1"))

	k8s.WaitLogsContains(nscPodNode, "nsm-init", "nsm client: initialization is completed successfully", defaultTimeout)
	logrus.Printf("VPN Gateway NSC started done: %v", time.Since(s1))

	var ipResponse = ""
	var routeResponse = ""
	var pingResponse = ""
	var errOut = ""
	var wgetResponse string

	if !k8s.UseIPv6() {
		ipResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr")
	} else {
		ipResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "-6", "addr")
	}
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	logrus.Printf("NSC IP status Ok")

	g.Expect(strings.Contains(ipResponse, srcIP)).To(Equal(true))
	g.Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

	if !k8s.UseIPv6() {
		routeResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "route")
	} else {
		routeResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "-6", "route")
	}
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	logrus.Printf("NSC Route status, Ok")

	g.Expect(strings.Contains(routeResponse, "nsm")).To(Equal(true))
	for i := 1; i <= 1; i++ {
		pingResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, pingCommand, dstIP, "-A", "-c", "10")
		g.Expect(err).To(BeNil())
		g.Expect(strings.Contains(pingResponse, "10 packets received")).To(Equal(true))
		logrus.Printf("VPN NSC Ping succeeded:%s", pingResponse)

		_, wgetResponse, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "wget", "-O", "/dev/null", "--timeout", "3", "http://"+dstIP+":80")
		g.Expect(err).To(BeNil())
		g.Expect(strings.Contains(wgetResponse, "100% |***")).To(Equal(true))
		logrus.Printf("%d VPN NSC wget request succeeded: %s", i, wgetResponse)

		_, wgetResponse, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "wget", "-O", "/dev/null", "--timeout", "3", "http://"+dstIP+":8080")
		g.Expect(err).To(Not(BeNil()))
		g.Expect(strings.Contains(wgetResponse, "download timed out")).To(Equal(true))
		logrus.Printf("%d VPN NSC wget request succeeded: %s", i, wgetResponse)
	}
}
