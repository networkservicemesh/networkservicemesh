package integration

import (
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"os"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSCAndICMPWireguard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMPWireguard(t, 2, false, false, "WIREGUARD")
}

func testNSCAndICMPWireguard(t *testing.T, nodesCount int, useWebhook, disableVHost bool, remoteMechanism string) {
	g := NewWithT(t)

	err := os.Setenv(pods.EnvForwardingPlane, pods.EnvForwardingPlaneKernel)
	g.Expect(err).To(BeNil())

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	//defer kubetest.MakeLogsSnapshot(k8s, t)
	g.Expect(err).To(BeNil())

	if useWebhook {
		awc, awDeployment, awService := kubetest.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace(), defaultTimeout)
		defer kubetest.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())
	}

	config := []*pods.NSMgrPodConfig{}
	for i := 0; i < nodesCount; i++ {
		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Variables[remote.PreferredRemoteMechanism.Name()] = remoteMechanism
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.ForwarderVariables = kubetest.DefaultForwarderVariables(pods.EnvForwardingPlaneKernel)
		if disableVHost {
			cfg.ForwarderVariables["FORWARDER_ALLOW_VHOST"] = "false"
		}
		config = append(config, cfg)
	}
	nodes_setup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	var nscPodNode *v1.Pod
	if useWebhook {
		nscPodNode = kubetest.DeployNSCWebhook(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	} else {
		nscPodNode = kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	}

	kubetest.CheckNSC(k8s, nscPodNode)
}