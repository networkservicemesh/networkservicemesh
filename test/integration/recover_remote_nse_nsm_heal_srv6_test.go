// +build srv6

package nsmd_integration_tests

import (
	"testing"
)

func TestNSMHealRemoteDieNSMD_NSE_SRv6(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSMHealRemoteDieNSMD_NSE(t, "SRV6")
}

func TestNSMHealRemoteDieNSMDSRv6(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSMHealRemoteDieNSMD(t, "SRV6")
}
