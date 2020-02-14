// +build recover

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSMHealRemoteDieNSMD_NSE(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSMHealRemoteDieNSMD_NSE(t, "VXLAN")
}

func TestNSMHealRemoteDieNSMD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSMHealRemoteDieNSMD(t, "VXLAN")
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
	defer k8s.SaveTestArtifacts(t)

	// Deploy open tracing to see what happening.
	//_, err = kubetest.SetupNodes(k8s, 2, defaultTimeout)
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

	kubetest.CheckNSC(k8s, nscPodNode)
}
