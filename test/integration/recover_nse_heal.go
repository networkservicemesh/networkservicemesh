// +build recover srv6

package nsmd_integration_tests

import (
	"strings"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSEHeal(t *testing.T, nodesCount int, affinity map[string]int, fixture kubetest.TestingPodFixture, remoteMechanism string) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	// Deploy open tracing to see what happening.
	config := []*pods.NSMgrPodConfig{}
	for i := 0; i < nodesCount; i++ {
		cfg := &pods.NSMgrPodConfig{
			Namespace:          k8s.GetK8sNamespace(),
			Variables:          pods.DefaultNSMD(),
			ForwarderVariables: kubetest.DefaultForwarderVariables(k8s.GetForwardingPlane()),
		}
		cfg.Variables[nsmd.NsmdPreferredRemoteMechanism] = remoteMechanism
		config = append(config, cfg)
	}
	nodesSetup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)

	// Run ICMP
	node := affinity["icmp-responder-nse-1"]
	nse1 := fixture.DeployNse(k8s, nodesSetup[node].Node, "icmp-responder-nse-1", defaultTimeout)

	nscPodNode := fixture.DeployNsc(k8s, nodesSetup[0].Node, "nsc-1", defaultTimeout)
	fixture.CheckNsc(k8s, nscPodNode)

	// Since all is fine now, we need to add new ICMP responder and delete previous one.
	node = affinity["icmp-responder-nse-2"]
	fixture.DeployNse(k8s, nodesSetup[node].Node, "icmp-responder-nse-2", defaultTimeout)

	logrus.Infof("Delete first NSE")
	k8s.DeletePods(nse1)

	logrus.Infof("Waiting for connection recovery...")

	k8s.WaitLogsContains(nodesSetup[0].Nsmd, "nsmd", "Heal: Connection recovered:", defaultTimeout)

	if len(nodesSetup) > 1 {
		l2, err := k8s.GetLogs(nodesSetup[1].Nsmd, "nsmd")
		g.Expect(err).To(BeNil())
		if strings.Contains(l2, "Forwarder request failed:") {
			logrus.Infof("Forwarder first attempt was failed: %v", l2)
		}
	}

	fixture.CheckNsc(k8s, nscPodNode)
}
