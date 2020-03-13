package utils

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/caddyserver/caddy/caddyfile"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	caddyfile_utils "github.com/networkservicemesh/networkservicemesh/utils/caddyfile"
)

const anyDomain = "."
const plugin = "forward"
const pluginAnyDomain = plugin + " " + anyDomain

//DNSConfigManager provides API for storing/deleting dnsConfigs. Can represent the configs in caddyfile format.
//Can be used from different goroutines
type DNSConfigManager struct {
	configs      sync.Map
	basicConfigs []*connectioncontext.DNSConfig
}

// NewDNSConfigManagerFromPath returns new dns config manager based on exist Caddyfile
func NewDNSConfigManagerFromPath(p string) (*DNSConfigManager, error) {
	f, err := os.Open(filepath.Clean(p))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Errorf("An error during close caddyfile: %v", err)
		}
	}()
	return NewDNSConfigManager(parseDNSConfigsFromCaddyfile(p, f)...), nil
}

func parseDNSConfigsFromCaddyfile(p string, r io.Reader) []*connectioncontext.DNSConfig {
	_, name := path.Split(p)
	d := caddyfile.NewDispenser(name, r)
	var configs []*connectioncontext.DNSConfig
	config := new(connectioncontext.DNSConfig)
	config.SearchDomains = d.RemainingArgs()
	for {
		if d.Val() == plugin {
			config.DnsServerIps = d.RemainingArgs()[1:] // skip dot
			configs = append(configs, config)
			config = new(connectioncontext.DNSConfig)
			for d.Next() {
				config.SearchDomains = append([]string{d.Val()}, d.RemainingArgs()...)
				if d.NextBlock() {
					break
				}
			}
			continue
		}
		if !d.Next() {
			break
		}
	}
	return configs
}

//NewDNSConfigManager creates new config manager
func NewDNSConfigManager(basic ...*connectioncontext.DNSConfig) *DNSConfigManager {
	return &DNSConfigManager{
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
func (m *DNSConfigManager) Caddyfile(path string) caddyfile_utils.Caddyfile {
	file := caddyfile_utils.NewCaddyfile(path)
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
	// NOTE discuss with Coredns about the relaod plugin improvements
	file.GetOrCreate(anyDomain).Write("reload 2s")
	return file
}

func (m *DNSConfigManager) writeDNSConfig(c caddyfile_utils.Caddyfile, config *connectioncontext.DNSConfig) {
	scopeName := strings.Join(config.SearchDomains, " ")
	if scopeName == "" {
		scopeName = anyDomain
	}

	ips := strings.Join(config.DnsServerIps, " ")
	if c.HasScope(scopeName) {
		fanoutIndex := 1
		ips += " " + c.GetOrCreate(scopeName).Records()[fanoutIndex].String()[len(pluginAnyDomain):]
		c.Remove(scopeName)
	}
	scope := c.WriteScope(scopeName)

	scope.Write("log").Write(fmt.Sprintf("%v %v", pluginAnyDomain, removeDuplicates(ips)))
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
