// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"testing"
)

func TestNSMDDP(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.Prepare("nsmd", "icmp")

	nodes := nsmd_test_utils.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMDPodConfig{})
	icmpPod := nsmd_test_utils.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nsmdName := nodes[0].Nsmd.Name
	k8s.DeletePods(nodes[0].Nsmd)
	k8s.DeletePods(icmpPod)

	nodes[0].Nsmd = k8s.CreatePod(pods.NSMDPodWithConfig(nsmdName, nodes[0].Node, &pods.NSMDPodConfig{})) // Recovery NSEs
	icmpPod = nsmd_test_utils.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-2", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())
}

func TestNSMDRecoverNSE(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.Prepare("nsmd", "icmp")

	nodes := nsmd_test_utils.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMDPodConfig{})
	icmpPod := nsmd_test_utils.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nsmdName := nodes[0].Nsmd.Name
	k8s.DeletePods(nodes[0].Nsmd)
	k8s.DeletePods(icmpPod)

	nodes[0].Nsmd = k8s.CreatePod(pods.NSMDPodWithConfig(nsmdName, nodes[0].Node, &pods.NSMDPodConfig{})) // Recovery NSEs
	icmpPod = nsmd_test_utils.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-2", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())
}
