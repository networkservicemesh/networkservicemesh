package utils_test

import (
	"github.com/ligato/networkservicemesh/examples/cmd/nsc/utils"
	. "github.com/onsi/gomega"
	"testing"
)

func TestSimpleConifg(t *testing.T) {
	RegisterTestingT(t)

	config := "nsm1:icmp-responder-nse, eth12:vpngatway"
	configMap := utils.ParseNetworkServices(config)

	Expect(configMap["icmp-responder-nse"]).To(Equal("nsm1"))
	Expect(configMap["vpngatway"]).To(Equal("eth12"))
}
