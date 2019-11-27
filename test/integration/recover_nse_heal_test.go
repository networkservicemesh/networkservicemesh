// +build recover

package nsmd_integration_tests

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
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
	}, kubetest.HealTestingPodFixture(g), "VXLAN")
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

func TestNSEHealLocalMemif(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSEHeal(t, 1, map[string]int{
		"icmp-responder-nse-1": 0,
		"icmp-responder-nse-2": 0,
	}, kubetest.VppAgentTestingPodFixture(g), "")
}
