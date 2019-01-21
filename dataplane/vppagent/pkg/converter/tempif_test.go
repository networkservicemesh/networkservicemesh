package converter_test

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/converter"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
)

func TestTempIf(t *testing.T) {
	tempIface := converter.TempIfName()
	fmt.Printf("tempIface: %s len(tempIface) %d\n", tempIface, len(tempIface))
	if len(tempIface) > connection.LinuxIfMaxLength {
		t.Errorf("%s is longer than %d", tempIface, connection.LinuxIfMaxLength)
	}
}
