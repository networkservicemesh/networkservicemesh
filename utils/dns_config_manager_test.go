package utils

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
)

func TestDnsConfigManagerCreation(t *testing.T) {
	assert := gomega.NewWithT(t)
	m := NewDNSConfigManager(*testBasicConfig())
	caddyfile := m.Caddyfile("test")
	assert.Expect(len(caddyfile.Records()) == 1).Should(gomega.BeTrue())
}

func TestDnsConfigManagerMergeConfigs(t *testing.T) {
	assert := gomega.NewWithT(t)
	m := NewDNSConfigManager(*testBasicConfig())
	other := *testBasicConfig()
	other.DnsServerIps = append(other.DnsServerIps, "192.168.0.1")
	m.Store("1", other)
	caddyfile := m.Caddyfile("test")
	assert.Expect(len(caddyfile.Records()) == 1).Should(gomega.BeTrue())
	assert.Expect(len(caddyfile.GetOrCreate(m.getBasicConfigScopeName()).Records()) == 2).Should(gomega.BeTrue())
	assert.Expect(caddyfile.GetOrCreate(m.getBasicConfigScopeName()).Records()[1].String()).Should(gomega.Equal("log"))
	assert.Expect(caddyfile.GetOrCreate(m.getBasicConfigScopeName()).Records()[2].String()).Should(gomega.Equal("fanout 127.0.0.1 192.168.0.1"))
}

func TestDnsConfigManagerStoreConfigs(t *testing.T) {
	assert := gomega.NewWithT(t)
	m := NewDNSConfigManager(*testBasicConfig())
	other := *testBasicConfig()
	other.DnsServerIps = append(other.DnsServerIps, "192.168.0.1")
	other.SearchDomains = append(other.SearchDomains, "other")
	m.Store("1", other)
	caddyfile := m.Caddyfile("test")
	assert.Expect(len(caddyfile.Records()) == 2).Should(gomega.BeTrue())
	assert.Expect(len(caddyfile.GetOrCreate("other").Records()) == 2).Should(gomega.BeTrue())
	assert.Expect(caddyfile.HasScope("other")).Should(gomega.BeTrue())
	assert.Expect(caddyfile.GetOrCreate("other").Records()[0].String()).Should(gomega.Equal("log"))
	assert.Expect(caddyfile.GetOrCreate("other").Records()[1].String()).Should(gomega.Equal("fanout 127.0.0.1 192.168.0.1"))
}

func TestDnsConfigManagerDeleteConfigs(t *testing.T) {
	assert := gomega.NewWithT(t)
	m := NewDNSConfigManager(*testBasicConfig())
	other := *testBasicConfig()
	other.DnsServerIps = append(other.DnsServerIps, "192.168.0.1")
	other.SearchDomains = append(other.SearchDomains, "other")
	m.Store("1", other)
	caddyfile := m.Caddyfile("test")
	assert.Expect(len(caddyfile.Records()) == 2).Should(gomega.BeTrue())
	m.Delete("1")
	caddyfile = m.Caddyfile("test")
	assert.Expect(len(caddyfile.Records()) == 1).Should(gomega.BeTrue())
}

func TestRemoveDuplicates(t *testing.T) {
	assert := gomega.NewWithT(t)
	r := removeDuplicates("a")
	assert.Expect(r).Should(gomega.Equal("a"))
	r = removeDuplicates("")
	assert.Expect(r).Should(gomega.Equal(""))
	r = removeDuplicates("aaa aaa bbb bbb aaa bbb ccc aa bb bb ee")
	assert.Expect(r).Should(gomega.Equal("aaa bbb ccc aa bb ee"))
}

func testBasicConfig() *connectioncontext.DNSConfig {
	return &connectioncontext.DNSConfig{
		DnsServerIps:  []string{"127.0.0.1"},
		SearchDomains: []string{},
	}
}
