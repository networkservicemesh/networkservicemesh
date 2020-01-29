package integration

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestKernelNSCAndICMPLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testKernelNSCAndICMP(t, 2, "")
}

func TestKernelNSCAndICMPRemoteVXLAN(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testKernelNSCAndICMP(t, 2, "VXLAN")
}

func TestKernelNSCAndICMPRemoteWireguard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testKernelNSCAndICMP(t, 2, "WIREGUARD")
}

func testKernelNSCAndICMP(t *testing.T, nodesCount int, remoteMechanism string) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	g.Expect(err).To(BeNil())

	defer k8s.Cleanup()
	defer kubetest.MakeLogsSnapshot(k8s, t)

	err = k8s.SetForwardingPlane(pods.EnvForwardingPlaneKernel)
	g.Expect(err).To(BeNil())

	config := []*pods.NSMgrPodConfig{}
	for i := 0; i < nodesCount; i++ {
		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Variables[remote.PreferredRemoteMechanism.Name()] = remoteMechanism
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.ForwarderVariables = kubetest.DefaultForwarderVariables(pods.EnvForwardingPlaneKernel)
		config = append(config, cfg)
	}
	nodesSetup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8s, nodesSetup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := kubetest.DeployNSC(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout)

	kubetest.CheckNSC(k8s, nscPodNode)
}
