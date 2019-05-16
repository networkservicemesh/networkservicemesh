// +build recover

package nsmd_integration_tests

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSMHealRemoteDieNSMD_NSE(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup := nsmd_test_utils.SetupNodesConfig(k8s, 2, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsm.NsmdHealDSTWaitTimeout:   "20", // 20 second delay, since we know both NSM and NSE will die and we need to go with different code branch.
				nsmd.NsmdDeleteLocalRegistry: "true",
			},
			Namespace:          k8s.GetK8sNamespace(),
			DataplaneVariables: nsmd_test_utils.DefaultDataplaneVariables(),
		},
		{
			Namespace:          k8s.GetK8sNamespace(),
			Variables:          pods.DefaultNSMD(),
			DataplaneVariables: nsmd_test_utils.DefaultDataplaneVariables(),
		},
	}, k8s.GetK8sNamespace())

	// Run ICMP on latest node
	icmpPod := nsmd_test_utils.DeployICMPWithConfig(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout, 30)

	nscPodNode := nsmd_test_utils.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *nsmd_test_utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	logrus.Infof("Delete Remote NSMD/ICMP responder NSE")
	k8s.DeletePods(nodes_setup[1].Nsmd)
	k8s.DeletePods(icmpPod)
	//k8s.DeletePods(nodes_setup[1].Nsmd, icmpPod)
	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", 60*time.Second)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	k8s.WaitLogsContains(nodes_setup[1].Nsmd, "nsmd", "NSM gRPC API Server: [::]:5001 is operational", defaultTimeout)
	k8s.WaitLogsContains(nodes_setup[1].Nsmd, "nsmdp", "ListAndWatch was called with", defaultTimeout)
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	failures = InterceptGomegaFailures(func() {
		// Restore ICMP responder pod.
		icmpPod = nsmd_test_utils.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-2", defaultTimeout)

		logrus.Infof("Waiting for connection recovery...")
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", 60*time.Second)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}

func TestNSMHealRemoteDieNSMD(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup := nsmd_test_utils.SetupNodes(k8s, 2, defaultTimeout)

	// Run ICMP on latest node
	icmpPod := nsmd_test_utils.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())

	nscPodNode := nsmd_test_utils.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *nsmd_test_utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

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

		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}


func TestNSMHealRemoteDieNSMDFakeEndpoint(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	nodes_setup := nsmd_test_utils.SetupNodes(k8s, 2, defaultTimeout)

	// Run ICMP on latest node
	icmpPod := nsmd_test_utils.DeployICMP(k8s, nodes_setup[1].Node, "icmp-responder-nse-1", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())

	nscPodNode := nsmd_test_utils.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	var nscInfo *nsmd_test_utils.NSCCheckInfo
	failures := InterceptGomegaFailures(func() {
		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)

	// Remember nse name
	_, nsm1RegistryClient, fwd1 := nsmd_test_utils.PrepareRegistryClients(k8s, nodes_setup[1].Nsmd)
	nseList, err := nsm1RegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	fwd1.Stop()

	Expect(err).To(BeNil())
	Expect(len(nseList.NetworkServiceEndpoints)).To(Equal(1))
	nseName := nseList.NetworkServiceEndpoints[0].EndpointName

	logrus.Infof("Delete Remote NSMD")
	k8s.DeletePods(nodes_setup[1].Nsmd)

	logrus.Infof("Waiting for NSE with network service")
	k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Waiting for NSE with network service icmp-responder", defaultTimeout)
	// Now are are in dataplane dead state, and in Heal procedure waiting for dataplane.
	nsmdName := fmt.Sprintf("nsmd-worker-recovered-%d", 1)

	logrus.Infof("Cleanup Endpoints CRDs...")
	k8s.CleanupEndpointsCRDs()

	nse2RegistryClient, nsm2RegistryClient, fwd2 := nsmd_test_utils.PrepareRegistryClients(k8s, nodes_setup[0].Nsmd)
	defer fwd2.Stop()

	_, err = nse2RegistryClient.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    "icmp-responder",
			Payload: "IP",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: "icmp-responder",
			EndpointName: nseName,
		},
	})
	Expect(err).To(BeNil())
	nseList, err = nsm2RegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	Expect(err).To(BeNil())
	Expect(len(nseList.NetworkServiceEndpoints)).To(Equal(1))

	logrus.Infof("Starting recovered NSMD...")
	startTime := time.Now()
	nodes_setup[1].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig(nsmdName, nodes_setup[1].Node, &pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	logrus.Printf("Started new NSMD: %v on node %s", time.Since(startTime), nodes_setup[1].Node.Name)

	failures = InterceptGomegaFailures(func() {
		logrus.Infof("Waiting for connection recovery...")
		k8s.WaitLogsContains(nodes_setup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)
		logrus.Infof("Waiting for connection recovery Done...")

		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	nsmd_test_utils.PrintErrors(failures, k8s, nodes_setup, nscInfo, t)
}