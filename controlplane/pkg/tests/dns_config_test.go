package tests

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
)

func TestDnsConfigValidateNil(t *testing.T) {
	gomega.RegisterTestingT(t)
	var config *connectioncontext.DNSConfig
	err := config.Validate()
	gomega.Expect(err.Error()).Should(gomega.Equal(connectioncontext.DNSConfigShouldNotBeNil))
}

func TestDnsConfigValidateNoRecords(t *testing.T) {
	gomega.RegisterTestingT(t)
	config := connectioncontext.DNSConfig{}
	err := config.Validate()
	gomega.Expect(err.Error()).Should(gomega.Equal(connectioncontext.DNSServerIpsShouldHaveRecords))
}
