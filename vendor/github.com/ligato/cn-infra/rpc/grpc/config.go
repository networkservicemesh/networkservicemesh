// Copyright (c) 2017 Cisco and/or its affiliates.
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

package grpc

import (
	"strconv"
	"strings"

	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/rpc/rest"
	"github.com/namsral/flag"
)

// PluginConfig tries :
// - to load flag <plugin-name>-port and then FixConfig() just in case
// - alternatively <plugin-name>-config and then FixConfig() just in case
// - alternatively DefaultConfig()
func PluginConfig(pluginCfg config.PluginConfig, cfg *Config, pluginName core.PluginName) error {
	portFlag := flag.Lookup(grpcPortFlag(pluginName))
	if portFlag != nil && portFlag.Value != nil && portFlag.Value.String() != "" && cfg != nil {
		cfg.Endpoint = rest.DefaultIP + ":" + portFlag.Value.String()
	}

	if pluginCfg != nil {
		_, err := pluginCfg.GetValue(cfg)
		if err != nil {
			return err
		}
	}

	FixConfig(cfg)

	return nil
}

// DefaultConfig returns new instance of config with default endpoint
func DefaultConfig() *Config {
	return &Config{} //endpoint is not set intentionally
}

// FixConfig fill default values for empty fields (does nothing yet)
func FixConfig(cfg *Config) {

}

// Config is a configuration for GRPC netListener
// It is meant to be extended with security (TLS...)
type Config struct {
	// Endpoint is an address of GRPC netListener
	Endpoint string

	// MaxMsgSize returns a ServerOption to set the max message size in bytes for inbound mesages.
	// If this is not set, gRPC uses the default 4MB.
	MaxMsgSize int

	// MaxConcurrentStreams returns a ServerOption that will apply a limit on the number
	// of concurrent streams to each ServerTransport.
	MaxConcurrentStreams uint32

	// Compression for inbound/outbound messages.
	// Supported only gzip.
	//TODO Compression string
	//TODO TLS/credentials
}

// GetPort parses suffix from endpoint & returns integer after last ":" (otherwise it returns 0)
func (cfg *Config) GetPort() int {
	if cfg.Endpoint != "" && cfg.Endpoint != ":" {
		index := strings.LastIndex(cfg.Endpoint, ":")
		if index >= 0 {
			port, err := strconv.Atoi(cfg.Endpoint[index+1:])
			if err == nil {
				return port
			}
		}
	}

	return 0
}

// DeclareGRPCPortFlag declares GRPC port (with usage & default value) a flag for a particular plugin name
func DeclareGRPCPortFlag(pluginName core.PluginName) {
	plugNameUpper := strings.ToUpper(string(pluginName))

	usage := "Configure Agent' " + plugNameUpper + " net listener (port & timeouts); also set via '" +
		plugNameUpper + config.EnvSuffix + "' env variable."
	flag.String(grpcPortFlag(pluginName), "", usage)
}

func grpcPortFlag(pluginName core.PluginName) string {
	return strings.ToLower(string(pluginName)) + "-port"
}
