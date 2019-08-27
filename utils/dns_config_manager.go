package utils

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/utils/caddyfile"
)

const anyDomain = "."

//DNSConfigManager provides API for storing/deleting dnsConfigs. Can represent the configs in caddyfile format.
//Can be used from different goroutines
type DNSConfigManager struct {
	configs     sync.Map
	basicConfig *connectioncontext.DNSConfig
	reloadTime  time.Duration
}

//NewDNSConfigManager creates new config manager
func NewDNSConfigManager(basic connectioncontext.DNSConfig, reloadTime time.Duration) *DNSConfigManager {
	return &DNSConfigManager{
		configs:     sync.Map{},
		basicConfig: &basic,
		reloadTime:  reloadTime,
	}
}

//Store stores new config with specific id
func (m *DNSConfigManager) Store(id string, config connectioncontext.DNSConfig) {
	m.configs.Store(id, &config)
}

//Delete deletes dns config by id
func (m *DNSConfigManager) Delete(id string) {
	m.configs.Delete(id)
}

//Caddyfile converts all configs to caddyfile
func (m *DNSConfigManager) Caddyfile(path string) caddyfile.Caddyfile {
	file := caddyfile.NewCaddyfile(path)
	m.writeDNSConfig(file, m.basicConfig)
	m.configs.Range(func(k, v interface{}) bool {
		config := v.(*connectioncontext.DNSConfig)
		m.writeDNSConfig(file, config)
		return true
	})
	return file
}

func (m *DNSConfigManager) getBasicConfigScopeName() string {
	r := strings.Join(m.basicConfig.SearchDomains, " ")
	if r == "" {
		return anyDomain
	}
	return r
}

func (m *DNSConfigManager) writeDNSConfig(c caddyfile.Caddyfile, config *connectioncontext.DNSConfig) {
	scopeName := strings.Join(config.SearchDomains, " ")
	if scopeName == "" {
		scopeName = anyDomain
	}

	ips := strings.Join(config.DnsServerIps, " ")

	if c.HasScope(scopeName) {
		fanoutIndex := 1
		ips += " " + c.GetOrCreate(scopeName).Records()[fanoutIndex].String()[len("fanout "):]
		c.Remove(scopeName)
	}
	scope := c.WriteScope(scopeName)

	scope.Write("log").Write(fmt.Sprintf("fanout %v", removeDuplicates(ips)))
}

func removeDuplicates(s string) string {
	if s == "" {
		return ""
	}
	words := strings.Split(s, " ")
	var result []string
	set := make(map[string]bool)
	for i := 0; i < len(words); i++ {
		if set[words[i]] {
			continue
		}
		set[words[i]] = true
		result = append(result, words[i])
	}
	return strings.Join(result, " ")
}
