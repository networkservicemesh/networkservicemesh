package main

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/utils"
	"github.com/networkservicemesh/networkservicemesh/utils/caddyfile"
	"github.com/sirupsen/logrus"
)

func main() {
	r := resolvConfFile{path: resolvConfFilePath}
	defaultDNSConfig := []*connectioncontext.DNSConfig{
		{
			//specific zone config
			DnsServerIps:  r.Nameservers(),
			SearchDomains: r.Searches(),
		},
		{
			//any zone config
			DnsServerIps: r.Nameservers(),
		},
	}
	properties := []resolvConfProperty{
		{nameserverProperty, []string{"127.0.0.1"}},
		{optionsProperty, r.Options()},
	}
	r.ReplaceProperties(properties)
	m := utils.NewDNSConfigManager(defaultDNSConfig...)
	f := m.Caddyfile(caddyfile.ParseCorefilePath())
	err := f.Save()
	if err != nil {
		logrus.Fatalf("An error during save caddy file %v", err)
	}
}
