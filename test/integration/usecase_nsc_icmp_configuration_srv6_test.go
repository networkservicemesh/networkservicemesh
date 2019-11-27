// +build srv6

package nsmd_integration_tests

import (
	"testing"
)

func TestNSCAndICMPRemoteSRv6(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, false, false, "SRV6")
}

func TestNSCAndICMPWebhookRemoteSRv6(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, true, false, "SRV6")
}

func TestNSCAndICMPRemoteVethSRv6(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, false, true, "SRV6")
}
