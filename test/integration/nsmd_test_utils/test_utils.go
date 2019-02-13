package nsmd_test_utils

import (
	"fmt"
	"time"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"strings"
	"testing"
)

const defaultTimeout = 60 * time.Second

type NodeConf struct {
	Nsmd      *v1.Pod
	Dataplane *v1.Pod
	Node      *v1.Node
}

func SetupNodes(k8s *kube_testing.K8s, nodesCount int) []*NodeConf {
	nodes := k8s.GetNodesWait(nodesCount, defaultTimeout)
	Expect(len(nodes) >= nodesCount).To(Equal(true),
		"At least one kubernetes node are required for this test")

	confs := []*NodeConf{}
	for i := 0; i < nodesCount; i++ {
		nsmd, dataplane := deployNsmdAndDataplane(k8s, &nodes[i])
		confs = append(confs, &NodeConf{
			Nsmd:      nsmd,
			Dataplane: dataplane,
			Node:      &nodes[i],
		})
	}
	return confs
}

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
	k8s.WaitLogsContains(nsmd, "nsmd", "NSM gRPC API Server: [::]:5001 is operational", defaultTimeout)
	k8s.WaitLogsContains(nsmd, "nsmdp", "ListAndWatch was called with", defaultTimeout)

	return
}

func DeployIcmp(k8s *kube_testing.K8s, node *v1.Node, name string) (icmp *v1.Pod) {
	startTime := time.Now()

	logrus.Infof("Starting ICMP Responder NSE on node: %s", node.Name)
	icmp = k8s.CreatePod(pods.ICMPResponderPod(name, node,
		map[string]string{
			"ADVERTISE_NSE_NAME":   "icmp-responder",
			"ADVERTISE_NSE_LABELS": "app=icmp",
			"IP_ADDRESS":           "10.20.1.0/24",
		},
	))
	Expect(icmp.Name).To(Equal(name))

	k8s.WaitLogsContains(icmp, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", defaultTimeout)

	logrus.Printf("ICMP Responder %v started done: %v", name, time.Since(startTime))
	return icmp
}

func DeployNsc(k8s *kube_testing.K8s, node *v1.Node, name string) (nsc *v1.Pod) {
	startTime := time.Now()

	logrus.Infof("Starting NSC %s on node: %s", name, node.Name)
	nsc = k8s.CreatePod(pods.NSCPod(name, node,
		map[string]string{
			"OUTGOING_NSC_LABELS": "app=icmp",
			"OUTGOING_NSC_NAME":   "icmp-responder",
		},
	))
	Expect(nsc.Name).To(Equal(name))

	k8s.WaitLogsContains(nsc, "nsc", "nsm client: initialization is completed successfully", defaultTimeout)
	logrus.Printf("NSC started done: %v", time.Since(startTime))
	return nsc
}

func PrintLogs(k8s *kube_testing.K8s, nodesSetup []*NodeConf) {
	for k := 0; k < len(nodesSetup); k++ {
		nsmdPod := nodesSetup[k].Nsmd
		nsmdLogs, _ := k8s.GetLogs(nsmdPod, "nsmd")
		logrus.Errorf("===================== NSMD %d output since test is failing %v\n=====================", k, nsmdLogs)

		nsmdk8sLogs, _ := k8s.GetLogs(nsmdPod, "nsmd-k8s")
		logrus.Errorf("===================== NSMD K8S %d output since test is failing %v\n=====================", k, nsmdk8sLogs)

		nsmdpLogs, _ := k8s.GetLogs(nsmdPod, "nsmdp")
		logrus.Errorf("===================== NSMD K8S %d output since test is failing %v\n=====================", k, nsmdpLogs)

		dataplaneLogs, _ := k8s.GetLogs(nodesSetup[k].Dataplane, "")
		logrus.Errorf("===================== Dataplane %d output since test is failing %v\n=====================", k, dataplaneLogs)
	}
}

type NSCCheckInfo struct {
	ipResponse string
	routeResponse string
	pingResponse string
	errOut string
}

func (info *NSCCheckInfo) PrintLogs() {
	logrus.Errorf("===================== NSC IP Addr %v\n=====================", info.ipResponse)
	logrus.Errorf("===================== NSC IP Route %v\n=====================", info.routeResponse)
	logrus.Errorf("===================== NSC IP PING %v\n=====================", info.pingResponse)
	logrus.Errorf("===================== NSC errOut %v\n=====================", info.errOut)
}

func CheckNSC(k8s *kube_testing.K8s, t *testing.T, nscPodNode *v1.Pod) *NSCCheckInfo {
	var err error
	info := &NSCCheckInfo{}
	info.ipResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr")
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))
	logrus.Printf("NSC IP status Ok")

	Expect(strings.Contains(info.ipResponse, "10.20.1.1")).To(Equal(true))
	Expect(strings.Contains(info.ipResponse, "nsm")).To(Equal(true))

	info.routeResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "route")
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))
	logrus.Printf("NSC Route status, Ok")

	Expect(strings.Contains(info.routeResponse, "8.8.8.8")).To(Equal(true))
	Expect(strings.Contains(info.routeResponse, "nsm")).To(Equal(true))

	info.pingResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ping", "10.20.1.2", "-c", "5")
	Expect(err).To(BeNil())
	Expect(strings.Contains(info.pingResponse, "5 packets transmitted, 5 packets received, 0% packet loss")).To(Equal(true))
	logrus.Printf("NSC Ping is success:%s", info.pingResponse)
	return info
}
