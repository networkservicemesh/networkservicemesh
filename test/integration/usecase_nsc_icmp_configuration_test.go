// +build usecase

package nsmd_integration_tests

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSCAndICMPLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, false, false)
}

func TestNSCAndICMPRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, false, false)
}

func TestNSCAndICMPWebhookLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, true, false)
}

func TestNSCAndICMPWebhookRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, true, false)
}

func TestNSCAndICMPLocalVeth(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, false, true)
}

func TestNSCAndICMPRemoteVeth(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, false, true)
}

func TestNSCAndICMPNeighbors(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	defer kubetest.ShowLogs(k8s, t)
	Expect(err).To(BeNil())

	nodes_setup, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())
	_ = kubetest.DeployNeighborNSE(k8s, nodes_setup[0].Node, "icmp-responder-nse-1", defaultTimeout)
	nsc := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)

	pingCommand := "ping"
	pingIP := "172.16.1.2"
	arpCommand := []string{"arp", "-a"}
	if k8s.UseIPv6() {
		pingCommand = "ping6"
		pingIP = "100::2"
		arpCommand = []string{"ip", "-6", "neigh", "show"}
	}
	pingResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, pingCommand, pingIP, "-A", "-c", "5")
	Expect(err).To(BeNil())
	Expect(errOut).To(Equal(""))
	Expect(strings.Contains(pingResponse, "100% packet loss")).To(Equal(false))

	nsc2 := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-2", defaultTimeout)
	arpResponse, errOut, err := k8s.Exec(nsc2, nsc.Spec.Containers[0].Name, arpCommand...)
	Expect(err).To(BeNil())
	Expect(errOut).To(Equal(""))
	Expect(strings.Contains(arpResponse, pingIP)).To(Equal(true))
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSCAndICMP(t *testing.T, nodesCount int, useWebhook bool, disableVHost bool) {
	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	defer kubetest.ShowLogs(k8s, t)
	Expect(err).To(BeNil())

	if useWebhook {
		awc, awDeployment, awService := kubetest.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace(), defaultTimeout)
		defer kubetest.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())
	}

	config := []*pods.NSMgrPodConfig{}
	for i := 0; i < nodesCount; i++ {
		cfg := &pods.NSMgrPodConfig{
			Variables: pods.DefaultNSMD(),
		}
		cfg.Namespace = k8s.GetK8sNamespace()
		cfg.DataplaneVariables = kubetest.DefaultDataplaneVariables(k8s.GetForwardingPlane())
		if disableVHost {
			cfg.DataplaneVariables["DATAPLANE_ALLOW_VHOST"] = "false"
		}
		config = append(config, cfg)
	}
	nodes_setup, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())
	Expect(err).To(BeNil())

	// Run ICMP on latest node
	_ = kubetest.DeployICMP(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	var nscPodNode *v1.Pod
	if useWebhook {
		nscPodNode = kubetest.DeployNSCWebhook(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	} else {
		nscPodNode = kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	}
	var nscInfo *kubetest.NSCCheckInfo

	failures := InterceptGomegaFailures(func() {
		nscInfo = kubetest.CheckNSC(k8s, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)

		nscInfo.PrintLogs()

		t.Fail()
	}
}
