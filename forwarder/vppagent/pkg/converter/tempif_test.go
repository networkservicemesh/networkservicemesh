// +build unit_test

package converter_test

import (
	"fmt"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"

	"github.com/networkservicemesh/networkservicemesh/forwarder/vppagent/pkg/converter"
)

func TestTempIf(t *testing.T) {
	tempIface := converter.TempIfName()
	fmt.Printf("tempIface: %s len(tempIface) %d\n", tempIface, len(tempIface))
	if len(tempIface) > kernel.LinuxIfMaxLength {
		t.Errorf("%s is longer than %d", tempIface, kernel.LinuxIfMaxLength)
	}
}
