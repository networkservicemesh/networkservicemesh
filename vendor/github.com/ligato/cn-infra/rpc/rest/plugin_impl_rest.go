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
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/utils/safeclose"
	"github.com/unrolled/render"
)

const (
	// DefaultHTTPPort is used during HTTP server startup unless different port was configured
	DefaultHTTPPort = "9191"
	// DefaultIP 0.0.0.0
	DefaultIP = "0.0.0.0"
	// DefaultEndpoint 0.0.0.0:9191
	DefaultEndpoint = DefaultIP + ":" + DefaultHTTPPort
)

// Plugin struct holds all plugin-related data.
type Plugin struct {
	Deps
	*Config

	// Used mainly for testing purposes
	listenAndServe ListenAndServe

	server    io.Closer
	mx        *mux.Router
	formatter *render.Render
}

// Deps lists the dependencies of the Rest plugin.
type Deps struct {
	Log                 logging.PluginLogger //inject
	PluginName          core.PluginName      //inject
	config.PluginConfig                      //inject
}

// Init is the plugin entry point called by Agent Core
// - It prepares Gorilla MUX HTTP Router
func (plugin *Plugin) Init() (err error) {
	if plugin.Config == nil {
		plugin.Config = DefaultConfig()
	}
	if err := PluginConfig(plugin.Deps.PluginConfig, plugin.Config, plugin.Deps.PluginName); err != nil {
		return err
	}

	plugin.mx = mux.NewRouter()
	plugin.formatter = render.New(render.Options{
		IndentJSON: true,
	})

	return err
}

// RegisterHTTPHandler registers HTTP <handler> at the given <path>.
func (plugin *Plugin) RegisterHTTPHandler(path string,
	handler func(formatter *render.Render) http.HandlerFunc,
	methods ...string) *mux.Route {
	plugin.Log.Debug("Register handler ", path)

	return plugin.mx.HandleFunc(path, handler(plugin.formatter)).Methods(methods...)
}

// GetPort returns plugin configuration port
func (plugin *Plugin) GetPort() int {
	if plugin.Config != nil {
		return plugin.Config.GetPort()
	}
	return 0
}

// AfterInit starts the HTTP server.
func (plugin *Plugin) AfterInit() (err error) {
	cfgCopy := *plugin.Config

	if plugin.listenAndServe != nil {
		plugin.server, err = plugin.listenAndServe(cfgCopy, plugin.mx)
	} else {
		plugin.Log.Info("Listening on http://", cfgCopy.Endpoint)
		plugin.server, err = ListenAndServeHTTP(cfgCopy, plugin.mx)
	}

	return err
}

// Close stops the HTTP server.
func (plugin *Plugin) Close() error {
	_, err := safeclose.CloseAll(plugin.server)
	return err
}

// String returns plugin name (if not set defaults to "HTTP")
func (plugin *Plugin) String() string {
	if plugin.Deps.PluginName != "" {
		return string(plugin.Deps.PluginName)
	}
	return "HTTP"
}
