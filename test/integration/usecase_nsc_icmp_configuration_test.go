// +build usecase

package nsmd_integration_tests

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestNSCAndICMPLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, false, false, "")
}

func TestNSCAndICMPRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, false, false, "VXLAN")
}

func TestNSCAndICMPWebhookLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, true, false, "")
}

func TestNSCAndICMPWebhookRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, true, false, "SRV6")
}

func TestNSCAndICMPLocalVeth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, false, true, "")
}

func TestNSCAndICMPRemoteVeth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, false, true, "VXLAN")
}

func TestNSCAndICMPNeighbors(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	defer kubetest.MakeLogsSnapshot(k8s, t)
	g.Expect(err).To(BeNil())

	nodes_setup, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())
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
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	g.Expect(strings.Contains(pingResponse, "100% packet loss")).To(Equal(false))

	nsc2 := kubetest.DeployNSC(k8s, nodes_setup[0].Node, "nsc-2", defaultTimeout)
	arpResponse, errOut, err := k8s.Exec(nsc2, nsc.Spec.Containers[0].Name, arpCommand...)
	g.Expect(err).To(BeNil())
	g.Expect(errOut).To(Equal(""))
	g.Expect(strings.Contains(arpResponse, pingIP)).To(Equal(true))
}
