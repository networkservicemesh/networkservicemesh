// +build recover

package nsmd_integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSMHealLocalDieNSMD(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)

	// Deploy open tracing to see what happening.
	nodes_setup, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := kubetest.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)
	g.Expect(icmpPod).ToNot(BeNil())

	nscPodNode := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)

	kubetest.CheckNSC(k8s, nscPodNode)

	logrus.Infof("Delete Local NSMD")
	k8s.DeletePods(nodes_setup[0].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[1].Nsmd, "nsmd", "NSM: Remote opened connection is not monitored and put into Healing state", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("%s-recovered", nodes_setup[0].Nsmd.Name)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[0].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[0].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[0].Node.Name)

	logrus.Infof("Waiting for connection recovery...")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	logrus.Infof("Waiting for connection recovery Done...")

	kubetest.HealNscChecker(k8s, nscPodNode)
}

func TestNSMHealLocalDieNSMDOneNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testNSMHealLocalDieNSMDOneNode(t, kubetest.DeployNSC, kubetest.DeployICMP, kubetest.CheckNSC, false)
}

func TestNSMHealLocalDieNSMDOneNodeMemif(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testNSMHealLocalDieNSMDOneNode(t, kubetest.DeployVppAgentNSC, kubetest.DeployVppAgentICMP, kubetest.CheckVppAgentNSC, false)
}

func TestNSMHealLocalDieNSMDOneNodeCleanedEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testNSMHealLocalDieNSMDOneNode(t, kubetest.DeployNSC, kubetest.DeployICMP, kubetest.CheckNSC, true)
}

func testNSMHealLocalDieNSMDOneNode(t *testing.T, deployNsc, deployNse kubetest.PodSupplier, nscCheck kubetest.NscChecker, cleanupEndpointsCRDs bool) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())
	kubetest.ShowLogs(k8s, t)
	// Deploy open tracing to see what happening.
	nodes_setup, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := deployNse(k8s, nodes_setup[0].Node, "icmp-responder-nse-1", defaultTimeout)
	g.Expect(icmpPod).ToNot(BeNil())

	nscPodNode := deployNsc(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	nscCheck(k8s, nscPodNode)

	logrus.Infof("Delete Local NSMD")
	k8s.DeletePods(nodes_setup[0].Nsmd)

	nsmdName := fmt.Sprintf("%s-recovered", nodes_setup[0].Nsmd.Name)

	if cleanupEndpointsCRDs {
		logrus.Infof("Cleanup Endpoints...")
		k8s.CleanupEndpointsCRDs()
	}

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[0].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[0].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[0].Node.Name)

	logrus.Infof("Waiting for connection recovery...")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	logrus.Infof("Waiting for connection recovery Done...")

	nscCheck(k8s, nscPodNode)
}

func TestNSMHealLocalDieNSMDOneNodeFakeEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testNSMHealLocalDieNSMDTwoNodes(t, kubetest.DeployNSC, kubetest.DeployICMP, kubetest.CheckNSC)
}

func testNSMHealLocalDieNSMDTwoNodes(t *testing.T, deployNsc, deployNse kubetest.PodSupplier, nscCheck kubetest.NscChecker) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())

	nodes_setup, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := deployNse(k8s, nodes_setup[0].Node, "icmp-responder-nse-1", defaultTimeout)
	g.Expect(icmpPod).ToNot(BeNil())

	nscPodNode := deployNsc(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	nscCheck(k8s, nscPodNode)

	// Remember nse name
	_, nsm1RegistryClient, fwd1Close := kubetest.PrepareRegistryClients(k8s, nodes_setup[0].Nsmd)
	nseList, err := nsm1RegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	fwd1Close()

	g.Expect(err).To(BeNil())
	g.Expect(len(nseList.NetworkServiceEndpoints)).To(Equal(1))
	nseName := nseList.NetworkServiceEndpoints[0].EndpointName

	logrus.Info(nseName)
	logrus.Infof("Delete Local NSMD")
	k8s.DeletePods(nodes_setup[0].Nsmd)

	nsmdName := fmt.Sprintf("%s-recovered", nodes_setup[0].Nsmd.Name)

	logrus.Infof("Cleanup Endpoints CRDs...")
	k8s.CleanupEndpointsCRDs()

	nse2RegistryClient, nsm2RegistryClient, fwd2Close := kubetest.PrepareRegistryClients(k8s, nodes_setup[1].Nsmd)
	defer fwd2Close()

	_, err = nse2RegistryClient.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    "icmp-responder",
			Payload: "IP",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: "icmp-responder",
			EndpointName:       nseName,
		},
	})
	g.Expect(err).To(BeNil())
	nseList2, err := nsm2RegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	g.Expect(err).To(BeNil())
	g.Expect(len(nseList2.NetworkServiceEndpoints)).To(Equal(1))

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[0].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[0].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[0].Node.Name)

	logrus.Infof("Waiting for connection recovery...")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	logrus.Infof("Waiting for connection recovery Done...")

	nscCheck(k8s, nscPodNode)
}
