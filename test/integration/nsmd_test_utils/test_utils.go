package nsmd_test_utils

import (
	"fmt"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
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

func DeployIcmp(k8s *kube_testing.K8s, node *v1.Node) (icmp *v1.Pod) {
	startTime := time.Now()

	logrus.Infof("Starting ICMP Responder NSE on node: %s", node.Name)
	icmp = k8s.CreatePod(pods.ICMPResponderPod("icmp-responder-nse1", node,
		map[string]string{
			"ADVERTISE_NSE_NAME":   "icmp-responder",
			"ADVERTISE_NSE_LABELS": "app=icmp",
			"IP_ADDRESS":           "10.20.1.0/24",
		},
	))
	Expect(icmp.Name).To(Equal("icmp-responder-nse1"))

	k8s.WaitLogsContains(icmp, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", defaultTimeout)

	logrus.Printf("ICMP Responder started done: %v", time.Since(startTime))
	return
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
	return
}
