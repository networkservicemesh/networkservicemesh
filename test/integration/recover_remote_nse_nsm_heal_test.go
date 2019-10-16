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

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSMHealRemoteDieNSMD_NSE(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)
	// Deploy open tracing to see what happening.
	nodes_setup, err := kubetest.SetupNodesConfig(k8s, 2, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsm.NsmdHealDSTWaitTimeout:   "20", // 20 second delay, since we know both NSM and NSE will die and we need to go with different code branch.
				nsmd.NsmdDeleteLocalRegistry: "true",
			},
			Namespace:          k8s.GetK8sNamespace(),
			DataplaneVariables: kubetest.DefaultDataplaneVariables(k8s.GetForwardingPlane()),
		},
		{
			Namespace:          k8s.GetK8sNamespace(),
			Variables:          pods.DefaultNSMD(),
			DataplaneVariables: kubetest.DefaultDataplaneVariables(k8s.GetForwardingPlane()),
		},
	}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := kubetest.DeployICMPWithConfig(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout, 30)

	nscPodNode := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	kubetest.CheckNSC(k8s, nscPodNode)

	logrus.Infof("Delete Remote NSMD/ICMP responder NSE")
	k8s.DeletePods(nodes_setup[1].Nsmd)
	k8s.DeletePods(icmpPod)
	//k8s.DeletePods(nodes_setup[1].Nsmd, icmpPod)
	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", time.Minute)
	// Now are are in forwarder dead state, and in Heal procedure waiting for forwarder.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	_ = k8s.WaitLogsContainsRegex(nodes_setup[1].Nsmd, "nsmd", "NSM gRPC API Server: .* is operational", defaultTimeout)
	k8s.WaitLogsContains(nodes_setup[1].Nsmd, "nsmdp", "nsmdp: successfully started", defaultTimeout)
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	// Restore ICMP responder pod.
	icmpPod = kubetest.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-2", defaultTimeout)

	logrus.Infof("Waiting for connection recovery...")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	logrus.Infof("Waiting for connection recovery Done...")

	kubetest.HealNscChecker(k8s, nscPodNode)
}

func TestNSMHealRemoteDieNSMD(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := kubetest.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)
	g.Expect(icmpPod).ToNot(BeNil())

	nscPodNode := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	kubetest.CheckNSC(k8s, nscPodNode)

	logrus.Infof("Delete Remote NSMD")
	k8s.DeletePods(nodes_setup[1].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", defaultTimeout)
	// Now are are in forwarder dead state, and in Heal procedure waiting for forwarder.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	logrus.Infof("Waiting for connection recovery...")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	logrus.Infof("Waiting for connection recovery Done...")

	kubetest.HealNscChecker(k8s, nscPodNode)
}

func TestNSMHealRemoteDieNSMDFakeEndpoint(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)

	// Deploy open tracing to see what happening.
	nodesSetup, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := kubetest.DeployICMP(k8s, nodesSetup[1].Node, "icmp-responder-nse-1", defaultTimeout)
	g.Expect(icmpPod).ToNot(BeNil())

	nscPodNode := kubetest.DeployNSC(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout)
	kubetest.CheckNSC(k8s, nscPodNode)

	// Remember nse name
	_, nsm1RegistryClient, fwd1Close := kubetest.PrepareRegistryClients(k8s, nodesSetup[1].Nsmd)
	nseList, err := nsm1RegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	fwd1Close()

	g.Expect(err).To(BeNil())
	g.Expect(len(nseList.NetworkServiceEndpoints)).To(Equal(1))
	nseName := nseList.NetworkServiceEndpoints[0].GetName()

	logrus.Infof("Delete Remote NSMD")
	k8s.DeletePods(nodesSetup[1].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodesSetup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", defaultTimeout)
	// Now are are in forwarder dead state, and in Heal procedure waiting for forwarder.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Cleanup Endpoints CRDs...")
	k8s.CleanupEndpointsCRDs()

	nse2RegistryClient, nsm2RegistryClient, fwd2Close := kubetest.PrepareRegistryClients(k8s, nodesSetup[0].Nsmd)
	defer fwd2Close()

	_, err = nse2RegistryClient.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    "icmp-responder",
			Payload: "IP",
		},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:               nseName,
			NetworkServiceName: "icmp-responder",
		},
	})
	g.Expect(err).To(BeNil())
	nseList, err = nsm2RegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	g.Expect(err).To(BeNil())
	g.Expect(len(nseList.NetworkServiceEndpoints)).To(Equal(1))

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodesSetup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodesSetup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodesSetup[1].Node.Name)

	logrus.Infof("Waiting for connection recovery...")
	k8s.WaitLogsContains(nodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
	logrus.Infof("Waiting for connection recovery Done...")

	kubetest.HealNscChecker(k8s, nscPodNode)
}
