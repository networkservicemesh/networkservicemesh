package nsmd_test_utils

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

type NodeConf struct {
	Nsmd      *v1.Pod
	Dataplane *v1.Pod
	Node      *v1.Node
}

func SetupNodes(k8s *kube_testing.K8s, nodesCount int, timeout time.Duration) []*NodeConf {
	return SetupNodesConfig(k8s, nodesCount, timeout, []*pods.NSMDPodConfig{})
}
func SetupNodesConfig(k8s *kube_testing.K8s, nodesCount int, timeout time.Duration, conf []*pods.NSMDPodConfig) []*NodeConf {
	nodes := k8s.GetNodesWait(nodesCount, timeout)
	Expect(len(nodes) >= nodesCount).To(Equal(true),
		"At least one Kubernetes node is required for this test")

	confs := []*NodeConf{}
	for i := 0; i < nodesCount; i++ {
		startTime := time.Now()
		node := &nodes[i]
		nsmdName := fmt.Sprintf("nsmd-%s", node.Name)
		dataplaneName := fmt.Sprintf("nsmd-dataplane-%s", node.Name)
		var corePod *v1.Pod
		debug := false
		if i >= len(conf) {
			corePod = pods.NSMDPod(nsmdName, node)
		} else {
			if conf[i].Nsmd == pods.NSMDPodDebug || conf[i].NsmdK8s == pods.NSMDPodDebug || conf[i].NsmdP == pods.NSMDPodDebug {
				debug = true
			}
			corePod = pods.NSMDPodWithConfig(nsmdName, node, conf[i])
		}
 		corePods := k8s.CreatePods(corePod, pods.VPPDataplanePod(dataplaneName, node))
 		if debug {
 			podContainer := "nsmd"
			if conf[i].Nsmd == pods.NSMDPodDebug {
				podContainer = "nsmd"
			} else if conf[i].NsmdP == pods.NSMDPodDebug {
				podContainer = "nsmdp"
			}

			k8s.WaitLogsContains(corePod, podContainer, "API server listening at: [::]:40000", timeout)
			logrus.Infof("Debug devenv container is running. Please do\n make k8s-forward pod=%v port1=40000 port2=40000. And attach via debugger...", corePod.Name)
		}
		nsmd, dataplane, err := deployNsmdAndDataplane(k8s, &nodes[i], corePods, timeout)

		logrus.Printf("Started NSMD/Dataplane: %v on node %s", time.Since(startTime), node.Name)
		Expect(err).To(BeNil())
		confs = append(confs, &NodeConf{
			Nsmd:      nsmd,
			Dataplane: dataplane,
			Node:      &nodes[i],
		})
	}
	return confs
}


func deployNsmdAndDataplane(k8s *kube_testing.K8s, node *v1.Node, corePods []*v1.Pod, timeout time.Duration) (nsmd *v1.Pod, dataplane *v1.Pod, err error) {
	for _, pod := range corePods {
		if !k8s.IsPodReady(pod) {
			return nil, nil, fmt.Errorf("Pod %v is not ready...", pod.Name)
		}
	}
	nsmd = corePods[0]
	dataplane = corePods[1]

	Expect(nsmd.Name).To(Equal(corePods[0].Name))
	Expect(dataplane.Name).To(Equal(corePods[1].Name))

	failures := InterceptGomegaFailures(func() {
		k8s.WaitLogsContains(dataplane, "", "Sending MonitorMechanisms update", timeout)
		k8s.WaitLogsContains(nsmd, "nsmd", "NSM gRPC API Server: [::]:5001 is operational", timeout)
		k8s.WaitLogsContains(nsmd, "nsmdp", "ListAndWatch was called with", timeout)
	})
	if len(failures) > 0 {
		printNSMDLogs(k8s, nsmd, 0 )
		printDataplaneLogs(k8s, dataplane, 0)
	}
	err = nil
	return
}

func DeployICMP(k8s *kube_testing.K8s, node *v1.Node, name string, timeout time.Duration) (icmp *v1.Pod) {
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

	k8s.WaitLogsContains(icmp, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", timeout)

	logrus.Printf("ICMP Responder %v started done: %v", name, time.Since(startTime))
	return icmp
}

func DeployNSC(k8s *kube_testing.K8s, node *v1.Node, name string, timeout time.Duration, useWebhook bool) (nsc *v1.Pod) {
	startTime := time.Now()

	logrus.Infof("Starting NSC %s on node: %s", name, node.Name)
	if useWebhook {
		nsc = k8s.CreatePod(pods.NSCPodWebhook(name, node))
	} else {
		nsc = k8s.CreatePod(pods.NSCPod(name, node,
			map[string]string{
				"OUTGOING_NSC_LABELS": "app=icmp",
				"OUTGOING_NSC_NAME":   "icmp-responder",
			},
		))
	}

	Expect(nsc.Name).To(Equal(name))

	k8s.WaitLogsContains(nsc, "nsc", "nsm client: initialization is completed successfully", timeout)
	logrus.Printf("NSC started done: %v", time.Since(startTime))
	return nsc
}

func PrintLogs(k8s *kube_testing.K8s, nodesSetup []*NodeConf) {
	for k := 0; k < len(nodesSetup); k++ {
		nsmdPod := nodesSetup[k].Nsmd
		printNSMDLogs(k8s, nsmdPod, k)

		printDataplaneLogs(k8s, nodesSetup[k].Dataplane, k)
	}
}

func printDataplaneLogs(k8s *kube_testing.K8s, dataplane *v1.Pod, k int) {
	dataplaneLogs, _ := k8s.GetLogs(dataplane, "")
	logrus.Errorf("===================== Dataplane %d output since test is failing %v\n=====================", k, dataplaneLogs)
}

func printNSMDLogs(k8s *kube_testing.K8s, nsmdPod *v1.Pod, k int) {
	nsmdLogs, _ := k8s.GetLogs(nsmdPod, "nsmd")
	logrus.Errorf("===================== NSMD %d output since test is failing %v\n=====================", k, nsmdLogs)
	nsmdk8sLogs, _ := k8s.GetLogs(nsmdPod, "nsmd-k8s")
	logrus.Errorf("===================== NSMD K8S %d output since test is failing %v\n=====================", k, nsmdk8sLogs)
	nsmdpLogs, _ := k8s.GetLogs(nsmdPod, "nsmdp")
	logrus.Errorf("===================== NSMD K8P %d output since test is failing %v\n=====================", k, nsmdpLogs)
}

type NSCCheckInfo struct {
	ipResponse    string
	routeResponse string
	pingResponse  string
	errOut        string
}

func (info *NSCCheckInfo) PrintLogs() {
	logrus.Errorf("===================== NSC IP Addr %v\n=====================", info.ipResponse)
	logrus.Errorf("===================== NSC IP Route %v\n=====================", info.routeResponse)
	logrus.Errorf("===================== NSC IP PING %v\n=====================", info.pingResponse)
	logrus.Errorf("===================== NSC errOut %v\n=====================", info.errOut)
}

func CheckNSC(k8s *kube_testing.K8s, t *testing.T, nscPodNode *v1.Pod) *NSCCheckInfo {
	return CheckNSCConfig(k8s, t, nscPodNode, "10.20.1.1", "10.20.1.2")
}
func CheckNSCConfig(k8s *kube_testing.K8s, t *testing.T, nscPodNode *v1.Pod, checkIP string, pingIP string) *NSCCheckInfo {
	var err error
	info := &NSCCheckInfo{}
	info.ipResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr")
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))
	logrus.Printf("NSC IP status Ok")

	Expect(strings.Contains(info.ipResponse, checkIP)).To(Equal(true))
	Expect(strings.Contains(info.ipResponse, "nsm")).To(Equal(true))

	info.routeResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "route")
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))
	logrus.Printf("NSC Route status, Ok")

	Expect(strings.Contains(info.routeResponse, "8.8.8.8")).To(Equal(true))
	Expect(strings.Contains(info.routeResponse, "nsm")).To(Equal(true))

	info.pingResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ping", pingIP, "-c", "5")
	Expect(err).To(BeNil())
	Expect(strings.Contains(info.pingResponse, "5 packets transmitted, 5 packets received, 0% packet loss")).To(Equal(true))
	logrus.Printf("NSC Ping is success:%s", info.pingResponse)
	return info
}

func IsBrokeTestsEnabled() bool {
	_, ok := os.LookupEnv("BROKEN_TESTS_ENABLED")
	return ok
}
