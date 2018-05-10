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

package rest

import (
	"strconv"
	"strings"
	"time"

	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/core"
	"github.com/namsral/flag"
)

// PluginConfig tries :
// - to load flag <plugin-name>-port and then FixConfig() just in case
// - alternatively <plugin-name>-config and then FixConfig() just in case
// - alternatively DefaultConfig()
func PluginConfig(pluginCfg config.PluginConfig, cfg *Config, pluginName core.PluginName) (error) {
	portFlag := flag.Lookup(httpPortFlag(pluginName))
	if portFlag != nil && portFlag.Value != nil && portFlag.Value.String() != "" && cfg != nil {
		cfg.Endpoint = DefaultIP + ":" + portFlag.Value.String()
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
	return &Config{Endpoint: DefaultEndpoint}
}

// FixConfig fill default values for empty fields
func FixConfig(cfg *Config) {
	if cfg != nil && cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
}

// Config is a configuration for HTTP server
// It is meant to be extended with security (TLS...)
type Config struct {
	// Endpoint is an address of HTTP server
	Endpoint string

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration

	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers. The connection's read deadline is reset
	// after reading the headers and the Handler can decide what
	// is considered too slow for the body.
	ReadHeaderTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout time.Duration

	// MaxHeaderBytes controls the maximum number of bytes the
	// server will read parsing the request header's keys and
	// values, including the request line. It does not limit the
	// size of the request body.
	// If zero, DefaultMaxHeaderBytes is used.
	MaxHeaderBytes int
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

// DeclareHTTPPortFlag declares http port (with usage & default value) a flag for a particular plugin name
func DeclareHTTPPortFlag(pluginName core.PluginName, defaultPortOpts ...uint) {
	var defaultPort string
	if len(defaultPortOpts) > 0 {
		defaultPort = string(defaultPortOpts[0])
	} else {
		defaultPort = DefaultHTTPPort
	}

	plugNameUpper := strings.ToUpper(string(pluginName))

	usage := "Configure Agent' " + plugNameUpper + " server (port & timeouts); also set via '" +
		plugNameUpper + config.EnvSuffix + "' env variable."
	flag.String(httpPortFlag(pluginName), defaultPort, usage)
}

func httpPortFlag(pluginName core.PluginName) string {
	return strings.ToLower(string(pluginName)) + "-port"
}
