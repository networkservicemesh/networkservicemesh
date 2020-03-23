// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dnsconfig

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
const defaultPlugin = "forward"
const conflictResolverPlugin = "fanout"

//DNSConfigManager provides API for storing/deleting dnsConfigs. Can represent the configs in caddyfile format.
//Can be used from different goroutines
type Manager interface {
	Caddyfile(string) caddyfile_utils.Caddyfile
	Delete(string)
	Store(string, ...*connectioncontext.DNSConfig)
}

type manager struct {
	configs      sync.Map
	basicConfigs []*connectioncontext.DNSConfig
}

// NewManagerFromCaddyfile returns new dns config manager based on exist Caddyfile
func NewManagerFromCaddyfile(path string) (Manager, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Errorf("An error during close caddyfile: %v", err)
		}
	}()
	return NewManager(parseDNSConfigsFromCaddyfile(path, f)...), nil
}

func parseDNSConfigsFromCaddyfile(location string, reader io.Reader) []*connectioncontext.DNSConfig {
	_, name := path.Split(location)
	d := caddyfile.NewDispenser(name, reader)
	var configs []*connectioncontext.DNSConfig
	config := new(connectioncontext.DNSConfig)
	config.SearchDomains = d.RemainingArgs()
	for {
		if d.Val() == defaultPlugin || d.Val() == conflictResolverPlugin {
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

//NewManager creates new config manager
func NewManager(basic ...*connectioncontext.DNSConfig) Manager {
	return &manager{
		basicConfigs: basic,
	}
}

//Store stores new config with specific id
func (m *manager) Store(id string, configs ...*connectioncontext.DNSConfig) {
	m.configs.Store(id, configs)
}

//Delete deletes dns config by id
func (m *manager) Delete(id string) {
	m.configs.Delete(id)
}

//Caddyfile converts all configs to caddyfile
func (m *manager) Caddyfile(path string) caddyfile_utils.Caddyfile {
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
	// NOTE discuss with Coredns about the relaod defaultPlugin improvements
	file.GetOrCreate(anyDomain).Write("reload 2s")
	return file
}

func (m *manager) writeDNSConfig(c caddyfile_utils.Caddyfile, config *connectioncontext.DNSConfig) {
	scopeName := strings.Join(config.SearchDomains, " ")
	plugin := defaultPlugin
	if scopeName == "" {
		scopeName = anyDomain
	}
	ips := strings.Join(config.DnsServerIps, " ")
	if c.HasScope(scopeName) {
		fanoutIndex := 1
		pluginName := strings.Split(c.GetOrCreate(scopeName).Records()[fanoutIndex].String(), " ")[0]
		ips += " " + c.GetOrCreate(scopeName).Records()[fanoutIndex].String()[len(pluginName)+3:]
		c.Remove(scopeName)
		plugin = conflictResolverPlugin
	}
	scope := c.WriteScope(scopeName)

	scope.Write("log").Write(fmt.Sprintf("%v %v %v", plugin, anyDomain, removeDuplicates(ips)))
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
