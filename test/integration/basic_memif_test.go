// +build basic

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"strings"
	"testing"
	"time"
)

func TestSimpleMemifConnection(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.PrepareDefault()

	nodes := nsmd_test_utils.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{})

	envIcmp := map[string]string{
		"ADVERTISE_NSE_NAME":   "icmp-responder",
		"ADVERTISE_NSE_LABELS": "app=icmp",
		"IP_ADDRESS":           "10.20.1.0/24",
	}
	vppagentIcmp := k8s.CreatePod(pods.VppagentICMPResponderPod("vppagent-icmp-responder", nodes[0].Node, envIcmp))
	Expect(vppagentIcmp.Name).To(Equal("vppagent-icmp-responder"))

	envNsc := map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   "icmp-responder",
	}
	vppagentNsc := k8s.CreatePod(pods.VppagentNSC("vppagent-nsc", nodes[0].Node, envNsc))
	Expect(vppagentNsc.Name).To(Equal("vppagent-nsc"))

	nseAvailable := false
	attempts := 30
	for ; attempts > 0; <-time.Tick(300 * time.Millisecond) {
		response, _, _ := k8s.Exec(vppagentNsc, vppagentNsc.Spec.Containers[0].Name, "vppctl", "ping", "10.20.1.2", "repeat", "2")
		if response != "" && !strings.Contains(response, "100% packet loss") {
			nseAvailable = true
			logrus.Info("Ping successful")
			break
		}
		attempts--
	}
	Expect(nseAvailable).To(Equal(true))
}
