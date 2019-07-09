package corefile

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"strings"
)

const (
	defaultCoreFileLocation = "/etc/coredns/Corefile"
	anyDomain               = "."
	bindAddr                = "127.0.0.1:1"
)

//DNSConfigManager - Manager of DNS configs
type DNSConfigManager interface {
	UpdateDNSConfig(config *connectioncontext.DNSConfig) error
	RemoveDNSConfig(config *connectioncontext.DNSConfig) error
}

//NewDefaultDNSConfigManager - Creates new instance of DNSConfigManager with default Corefile path
func NewDefaultDNSConfigManager() (DNSConfigManager, error) {
	return NewDNSConfigManager(defaultCoreFileLocation)
}

//NewDNSConfigManager - Creates new instance of DNSConfigManager with custom Corefile path
func NewDNSConfigManager(coreFilePath string) (DNSConfigManager, error) {
	c := NewCorefile(coreFilePath)
	c.WriteScope(anyDomain + ":54").Write(fmt.Sprintf("bind %v", bindAddr)).Write("log").Write("reload 5s")
	err := c.Save()
	if err != nil {
		return nil, err
	}
	return &dnsConfigManager{
		corefile: c,
	}, nil
}

type dnsConfigManager struct {
	corefile Corefile
}

//UpdateDNSConfig - Updates Corefile with new DNSConfig
func (m *dnsConfigManager) UpdateDNSConfig(config *connectioncontext.DNSConfig) error {
	err := config.Validate()
	if err != nil {
		return err
	}
	domains := strings.Join(config.ResolvesDomains, " ")
	ips := strings.Join(config.DnsServerIps, " ")
	if domains == "" {
		domains = anyDomain
	}
	s := m.corefile.WriteScope(domains).Write(fmt.Sprintf("bind %v", bindAddr)).Write("log").Write("fanout . " + ips)
	if config.Prioritize {
		s.Prioritize()
	}
	return m.corefile.Save()
}

//RemoveDNSConfig - Removes DNSConfig from Corefile
func (m *dnsConfigManager) RemoveDNSConfig(config *connectioncontext.DNSConfig) error {
	err := config.Validate()
	if err != nil {
		return err
	}
	domains := strings.Join(config.ResolvesDomains, " ")
	if domains == "" {
		domains = anyDomain
	}
	m.corefile.Remove(domains)
	return m.corefile.Save()
}
