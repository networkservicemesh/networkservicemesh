package tests

import (
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/onsi/gomega"
)

func TestDnsConfigValidateNil(t *testing.T) {
	gomega.RegisterTestingT(t)
	var config *networkservice.DNSConfig
	err := config.Validate()
	gomega.Expect(err.Error()).Should(gomega.Equal(networkservice.DNSConfigShouldNotBeNil))
}

func TestDnsConfigValidateNoRecords(t *testing.T) {
	gomega.RegisterTestingT(t)
	config := networkservice.DNSConfig{}
	err := config.Validate()
	gomega.Expect(err.Error()).Should(gomega.Equal(networkservice.DNSServerIpsShouldHaveRecords))
}
