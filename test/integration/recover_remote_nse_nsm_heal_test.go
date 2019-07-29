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

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSMHealRemoteDieNSMD_NSE(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())
	defer kubetest.ShowLogs(k8s, t)
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
	Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := kubetest.DeployICMPWithConfig(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout, 30)

	nscPodNode := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *kubetest.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = kubetest.CheckNSC(k8s, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	kubetest.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	logrus.Infof("Delete Remote NSMD/ICMP responder NSE")
	k8s.DeletePods(nodes_setup[1].Nsmd)
	k8s.DeletePods(icmpPod)
	//k8s.DeletePods(nodes_setup[1].Nsmd, icmpPod)
	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", time.Minute)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	_ = k8s.WaitLogsContainsRegex(nodes_setup[1].Nsmd, "nsmd", "NSM gRPC API Server: .* is operational", defaultTimeout)
	k8s.WaitLogsContains(nodes_setup[1].Nsmd, "nsmdp", "nsmdp: successfully started", defaultTimeout)
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	failures = InterceptGomegaFailures(func() {
		// Restore ICMP responder pod.
		icmpPod = kubetest.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-2", defaultTimeout)

		logrus.Infof("Waiting for connection recovery...")
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", time.Minute)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = kubetest.HealNscChecker(k8s, nscPodNode)
	})
	kubetest.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}

func TestNSMHealRemoteDieNSMD(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := kubetest.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())

	nscPodNode := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *kubetest.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = kubetest.CheckNSC(k8s, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	kubetest.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	logrus.Infof("Delete Remote NSMD")
	k8s.DeletePods(nodes_setup[1].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	failures = InterceptGomegaFailures(func() {
		logrus.Infof("Waiting for connection recovery...")
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = kubetest.HealNscChecker(k8s, nscPodNode)
	})
	kubetest.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}

func TestNSMHealRemoteDieNSMDFakeEndpoint(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodesSetup, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())

	// Run ICMP on latest node
	icmpPod := kubetest.DeployICMP(k8s, nodesSetup[1].Node, "icmp-responder-nse-1", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())

	nscPodNode := kubetest.DeployNSC(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *kubetest.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = kubetest.CheckNSC(k8s, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	kubetest.PrintErrors(failures, k8s, nodesSetup, nscInfo, t)

	// Remember nse name
	_, nsm1RegistryClient, fwd1Close := kubetest.PrepareRegistryClients(k8s, nodesSetup[1].Nsmd)
	nseList, err := nsm1RegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	fwd1Close()

	Expect(err).To(BeNil())
	Expect(len(nseList.NetworkServiceEndpoints)).To(Equal(1))
	nseName := nseList.NetworkServiceEndpoints[0].EndpointName

	logrus.Infof("Delete Remote NSMD")
	k8s.DeletePods(nodesSetup[1].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodesSetup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
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
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: "icmp-responder",
			EndpointName:       nseName,
		},
	})
	Expect(err).To(BeNil())
	nseList, err = nsm2RegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	Expect(err).To(BeNil())
	Expect(len(nseList.NetworkServiceEndpoints)).To(Equal(1))

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodesSetup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodesSetup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodesSetup[1].Node.Name)

	failures = InterceptGomegaFailures(func() {
		logrus.Infof("Waiting for connection recovery...")
		k8s.WaitLogsContains(nodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = kubetest.HealNscChecker(k8s, nscPodNode)
	})
	kubetest.PrintErrors(failures, k8s, nodesSetup, nscInfo, t)
}
