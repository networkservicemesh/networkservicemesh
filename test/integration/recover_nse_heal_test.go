// +build recover_suite

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestNSEHealLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testNSEHeal(t, 1, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 0,
	}, kubetest.DefaultTestingPodFixture(g), "")
}

func TestNSEHealLocalToRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testNSEHeal(t, 2, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 1,
	}, kubetest.DefaultTestingPodFixture(g), "")
}

func TestNSEHealRemoteToLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testNSEHeal(t, 2, map[string]int{
		"icmp-responder-nse-1": 1,
		"icmp-responder-nse-2": 0,
	}, kubetest.DefaultTestingPodFixture(g), "VXLAN")
}

func TestNSEHealRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	g := NewWithT(t)

	testNSEHeal(t, 2, map[string]int{
		"icmp-responder-nse-1": 1,
		"icmp-responder-nse-2": 1,
	}, kubetest.DefaultTestingPodFixture(g), "")
}

func TestNSEHealLocalVpp(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 1, map[string]int{
		"vpp-agent-nse-1": 0,
		"vpp-agent-nse-2": 0,
	}, kubetest.VppAgentTestingPodFixture(g), "")
}

func TestNSEHealToLocalVpp(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 2, map[string]int{
		"vpp-agent-nse-1": 1,
		"vpp-agent-nse-2": 0,
	}, kubetest.VppAgentTestingPodFixture(g), "")
}

func TestNSEHealToRemoteVpp(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 2, map[string]int{
		"vpp-agent-nse-1": 0,
		"vpp-agent-nse-2": 1,
	}, kubetest.VppAgentTestingPodFixture(g), "")
}

func TestNSEHealRemoteVpp(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 2, map[string]int{
		"vpp-agent-nse-1": 1,
		"vpp-agent-nse-2": 1,
	}, kubetest.VppAgentTestingPodFixture(g), "")
}
