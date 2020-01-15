// +build single_cluster_suite

package nsmd_integration_tests

import (
	"testing"
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

	testNSCAndICMP(t, 2, true, false, "")
}
