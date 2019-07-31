// +build interdomain

package nsmd_integration_tests

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	nsapiv1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/crds"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

const (
	InterdomainPtNum = 5 // Number of Passthrough Endpoints to deploy
)

/* Disable Firewall Remote test while vxlan has vni conflict
func TestInterdomainVPNFirewallRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainVPN(t, InterdomainPtNum, 2, 1, map[string]int{
		"vppagent-firewall-nse-1":  1,
		"vppagent-passthrough-nse": 0,
		"vpn-gateway-nse-1":        0,
		"vpn-gateway-nsc-1":        0,
	}, false)
}
*/

func TestInterdomainVPNNSERemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainVPN(t, InterdomainPtNum, 2, 1, map[string]int{
		"vppagent-firewall-nse-1":  0,
		"vppagent-passthrough-nse": 0,
		"vpn-gateway-nse-1":        1,
		"vpn-gateway-nsc-1":        0,
	}, false)
}

func TestInterdomainVPNNSCRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testInterdomainVPN(t, InterdomainPtNum, 2, 1, map[string]int{
		"vppagent-firewall-nse-1":  0,
		"vppagent-passthrough-nse": 0,
		"vpn-gateway-nse-1":        0,
		"vpn-gateway-nsc-1":        1,
	}, false)
}

func testInterdomainVPN(t *testing.T, ptnum, clustersCount int, nodesCount int, affinity map[string]int, verbose bool) {
	g := NewWithT(t)

	k8ss := []*kubetest.ExtK8s{}
	clusterNodes := [][]v1.Node{}

	for i := 0; i < clustersCount; i++ {
		kubeconfig := os.Getenv(fmt.Sprintf("KUBECONFIG_CLUSTER_%d", i+1))
		g.Expect(len(kubeconfig)).ToNot(Equal(0))

		k8s, err := kubetest.NewK8sForConfig(g, true, kubeconfig)

		g.Expect(err).To(BeNil())

		config := []*pods.NSMgrPodConfig{}

		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.DataplaneVariables = kubetest.DefaultDataplaneVariables(k8s.GetForwardingPlane())

		config = append(config, cfg)

		nodesSetup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
		g.Expect(err).To(BeNil())
		defer kubetest.ShowLogs(k8s, t)

		k8ss = append(k8ss, &kubetest.ExtK8s{
			K8s:        k8s,
			NodesSetup: nodesSetup,
		})

		for j := 0; j < nodesCount; j++ {
			pnsmdName := fmt.Sprintf("pnsmgr-%s", nodesSetup[j].Node.Name)
			kubetest.DeployProxyNSMgr(k8s, nodesSetup[j].Node, pnsmdName, defaultTimeout)
		}

		serviceCleanup := kubetest.RunProxyNSMgrService(k8s)
		defer serviceCleanup()

		defer k8ss[i].K8s.Cleanup()

		nodes := k8s.GetNodesWait(nodesCount, defaultTimeout)
		if len(nodes) < nodesCount {
			logrus.Printf("At least one Kubernetes node is required for this test")
			g.Expect(len(nodes)).To(Equal(nodesCount))
			return
		}
		clusterNodes = append(clusterNodes, nodes)
	}

	nscCluster := affinity["vpn-gateway-nsc-1"]

	nseCluster := affinity["vpn-gateway-nse-1"]
	nseExternalIP, err := kubetest.GetNodeExternalIP(k8ss[nseCluster].NodesSetup[0].Node)
	if err != nil {
		nseExternalIP, err = kubetest.GetNodeInternalIP(k8ss[nseCluster].NodesSetup[0].Node)
		g.Expect(err).To(BeNil())
	}

	firewallCluster := affinity["vppagent-firewall-nse-1"]
	firewallExternalIP, err := kubetest.GetNodeExternalIP(k8ss[firewallCluster].NodesSetup[0].Node)
	if err != nil {
		firewallExternalIP, err = kubetest.GetNodeInternalIP(k8ss[firewallCluster].NodesSetup[0].Node)
		g.Expect(err).To(BeNil())
	}

	s1 := time.Now()

	{
		nscrd, err := crds.NewNSCRDWithConfig(k8ss[0].K8s.GetK8sNamespace(), os.Getenv("KUBECONFIG_CLUSTER_1"))
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
	}

	pingCommand := "ping"
	addressPool := "172.16.1.0/24"
	srcIP, dstIP := "172.16.1.1", "172.16.1.2"

	/* Change stuff related to IPv6 */
	if k8ss[0].K8s.UseIPv6() && k8ss[clustersCount].K8s.UseIPv6() {
		pingCommand = "ping6"
		addressPool = "100::/64"
		srcIP, dstIP = "100::1", "100::2"
	}

	nscOutgoingName := "secure-intranet-connectivity"
	if firewallCluster != nseCluster {
		nscOutgoingName = fmt.Sprintf("secure-intranet-connectivity@%s", nseExternalIP)
	}

	s1 = time.Now()
	logrus.Infof("Starting VPPAgent Firewall NSE on node: %d", firewallCluster)
	_, err = k8ss[firewallCluster].K8s.CreateConfigMap(pods.VppAgentFirewallNSEConfigMapICMPHTTP("vppagent-firewall-nse-1", k8ss[firewallCluster].K8s.GetK8sNamespace()))
	g.Expect(err).To(BeNil())
	vppagentFirewallNode := k8ss[firewallCluster].K8s.CreatePod(pods.VppAgentFirewallNSEPodWithConfigMap("vppagent-firewall-nse-1", &clusterNodes[firewallCluster][0],
		map[string]string{
			"ADVERTISE_NSE_NAME":   "secure-intranet-connectivity",
			"ADVERTISE_NSE_LABELS": "app=firewall",
			"OUTGOING_NSC_NAME":    nscOutgoingName,
			"OUTGOING_NSC_LABELS":  "app=firewall",
		},
	))
	g.Expect(vppagentFirewallNode.Name).To(Equal("vppagent-firewall-nse-1"))

	k8ss[firewallCluster].K8s.WaitLogsContains(vppagentFirewallNode, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", fastTimeout)

	logrus.Printf("VPN firewall started done: %v", time.Since(s1))

	for i := 1; i <= ptnum; i++ {
		s1 = time.Now()
		id := strconv.Itoa(i)
		passthroughCluster := affinity["vppagent-passthrough-nse"]
		logrus.Infof("Starting VPPAgent Passthrough NSE on node: %d", passthroughCluster)

		vppagentPassthroughNode := k8ss[passthroughCluster].K8s.CreatePod(pods.VppAgentFirewallNSEPod("vppagent-passthrough-nse-"+id, &clusterNodes[passthroughCluster][0],
			map[string]string{
				"ADVERTISE_NSE_NAME":   "secure-intranet-connectivity",
				"ADVERTISE_NSE_LABELS": "app=passthrough-" + id,
				"OUTGOING_NSC_NAME":    "secure-intranet-connectivity",
				"OUTGOING_NSC_LABELS":  "app=passthrough-" + id,
			},
		))
		g.Expect(vppagentPassthroughNode.Name).To(Equal("vppagent-passthrough-nse-" + id))

		k8ss[passthroughCluster].K8s.WaitLogsContains(vppagentPassthroughNode, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", fastTimeout)

		logrus.Printf("VPN passthrough started done: %v", time.Since(s1))
	}

	s1 = time.Now()
	logrus.Infof("Starting VPN Gateway NSE on node: %d", nseCluster)
	vpnGatewayPodNode := k8ss[nseCluster].K8s.CreatePod(pods.VPNGatewayNSEPod("vpn-gateway-nse-1", &clusterNodes[nseCluster][0],
		map[string]string{
			"ADVERTISE_NSE_NAME":   "secure-intranet-connectivity",
			"ADVERTISE_NSE_LABELS": "app=vpn-gateway",
			"IP_ADDRESS":           addressPool,
		},
	))
	g.Expect(vpnGatewayPodNode).ToNot(BeNil())
	g.Expect(vpnGatewayPodNode.Name).To(Equal("vpn-gateway-nse-1"))

	k8ss[nseCluster].K8s.WaitLogsContains(vpnGatewayPodNode, "vpn-gateway", "NSE: channel has been successfully advertised, waiting for connection from NSM...", fastTimeout)

	logrus.Printf("VPN Gateway started done: %v", time.Since(s1))

	s1 = time.Now()
	nscOutgoingName = "secure-intranet-connectivity"
	if firewallCluster != nscCluster {
		nscOutgoingName = fmt.Sprintf("secure-intranet-connectivity@%s", firewallExternalIP)
	}
	nscPodNode := k8ss[nscCluster].K8s.CreatePod(pods.NSCPod("vpn-gateway-nsc-1", &clusterNodes[nscCluster][0],
		map[string]string{
			"OUTGOING_NSC_NAME": nscOutgoingName,
		},
	))
	g.Expect(nscPodNode.Name).To(Equal("vpn-gateway-nsc-1"))

	k8ss[nscCluster].K8s.WaitLogsContains(nscPodNode, "nsm-init", "nsm client: initialization is completed successfully", defaultTimeout)
	logrus.Printf("VPN Gateway NSC started done: %v", time.Since(s1))

	var ipResponse = ""
	var routeResponse = ""
	var pingResponse = ""
	var errOut = ""
	var wgetResponse string

	if !k8ss[nscCluster].K8s.UseIPv6() {
		ipResponse, errOut, err = k8ss[nscCluster].K8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr")
	} else {
		ipResponse, errOut, err = k8ss[nscCluster].K8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "-6", "addr")
	}
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	logrus.Printf("NSC IP status Ok")

	g.Expect(strings.Contains(ipResponse, srcIP)).To(Equal(true))
	g.Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

	if !k8ss[nscCluster].K8s.UseIPv6() {
		routeResponse, errOut, err = k8ss[nscCluster].K8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "route")
	} else {
		routeResponse, errOut, err = k8ss[nscCluster].K8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "-6", "route")
	}
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	logrus.Printf("NSC Route status, Ok")

	g.Expect(strings.Contains(routeResponse, "nsm")).To(Equal(true))
	for i := 1; i <= 1; i++ {
		pingResponse, errOut, err = k8ss[nscCluster].K8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, pingCommand, dstIP, "-A", "-c", "10")
		g.Expect(err).To(BeNil())
		g.Expect(strings.Contains(pingResponse, "10 packets received")).To(Equal(true))
		logrus.Printf("VPN NSC Ping succeeded:%s", pingResponse)

		_, wgetResponse, err = k8ss[nscCluster].K8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "wget", "-O", "/dev/null", "--timeout", "3", "http://"+dstIP+":80")
		g.Expect(err).To(BeNil())
		g.Expect(strings.Contains(wgetResponse, "100% |***")).To(Equal(true))
		logrus.Printf("%d VPN NSC wget request succeeded: %s", i, wgetResponse)

		_, wgetResponse, err = k8ss[nscCluster].K8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "wget", "-O", "/dev/null", "--timeout", "3", "http://"+dstIP+":8080")
		g.Expect(err).To(Not(BeNil()))
		g.Expect(strings.Contains(wgetResponse, "download timed out")).To(Equal(true))
		logrus.Printf("%d VPN NSC wget request succeeded: %s", i, wgetResponse)
	}
}
