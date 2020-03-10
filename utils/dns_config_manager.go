package utils

import (
	"fmt"
	"strings"
	"sync"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/utils/caddyfile"
)

const anyDomain = "."
const corednsPlugin = "forward"

//DNSConfigManager provides API for storing/deleting dnsConfigs. Can represent the configs in caddyfile format.
//Can be used from different goroutines
type DNSConfigManager struct {
	configs      sync.Map
	basicConfigs []*connectioncontext.DNSConfig
}

//NewDNSConfigManager creates new config manager
func NewDNSConfigManager(basic ...*connectioncontext.DNSConfig) *DNSConfigManager {
	return &DNSConfigManager{
		configs:      sync.Map{},
		basicConfigs: basic,
	}
}

//Store stores new config with specific id
func (m *DNSConfigManager) Store(id string, configs ...*connectioncontext.DNSConfig) {
	m.configs.Store(id, configs)
}

//Delete deletes dns config by id
func (m *DNSConfigManager) Delete(id string) {
	m.configs.Delete(id)
}

//Caddyfile converts all configs to caddyfile
func (m *DNSConfigManager) Caddyfile(path string) caddyfile.Caddyfile {
	file := caddyfile.NewCaddyfile(path)
	for _, c := range m.basicConfigs {
		m.writeDNSConfig(file, c)
	}
	m.configs.Range(func(k, v interface{}) bool {
		configs := v.([]*connectioncontext.DNSConfig)
		for _, c := range configs {
			m.writeDNSConfig(file, c)
		}
		return true
	})
	// TODO discuss with Coredns about the relaod plugin improvements
	file.GetOrCreate(anyDomain).Write("reload 2s")
	return file
}

func (m *DNSConfigManager) writeDNSConfig(c caddyfile.Caddyfile, config *connectioncontext.DNSConfig) {
	scopeName := strings.Join(config.SearchDomains, " ")
	if scopeName == "" {
		scopeName = anyDomain
	}

	ips := strings.Join(config.DnsServerIps, " ")
	if c.HasScope(scopeName) {
		fanoutIndex := 1
		ips += " " + c.GetOrCreate(scopeName).Records()[fanoutIndex].String()[len(corednsPlugin):]
		c.Remove(scopeName)
	}
	scope := c.WriteScope(scopeName)

	scope.Write("log").Write(fmt.Sprintf("%v %v", corednsPlugin, removeDuplicates(ips)))
}

func removeDuplicates(s string) string {
	if s == "" {
		return ""
	}
	words := strings.Split(s, " ")
	var result []string
	set := make(map[string]bool)
	for i := 0; i < len(words); i++ {
		if words[i] == "" {
			continue
		}
		if set[words[i]] {
			continue
		}
		set[words[i]] = true
		result = append(result, words[i])
	}
	return strings.Join(result, " ")
}
