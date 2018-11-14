package vppagent_test

import (
	"fmt"
	"testing"

	"github.com/ligato/networkservicemesh/dataplane/vppagent/pkg/vppagent"
)

func TestTempIf(t *testing.T) {
	tempIface := vppagent.TempIfName()
	fmt.Printf("tempIface: %s len(tempIface) %d\n", tempIface, len(tempIface))
	if len(tempIface) > vppagent.LinuxIfMaxLength {
		t.Errorf("%s is longer than %d", tempIface, vppagent.LinuxIfMaxLength)
	}
}
