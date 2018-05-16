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
	"github.com/unrolled/render"
)

// ForkPlugin checks the configuration and based on this it
// delegates API calls to new instance or existing instance of HTTP server
type ForkPlugin struct {
	Deps ForkDeps
	*Config

	delegate  HTTPHandlers
	newPlugin *Plugin //set only if delegate != Deps.DefaultHTTP

	// Used mainly for testing purposes
	listenAndServe ListenAndServe

	server    io.Closer
	mx        *mux.Router
	formatter *render.Render
}

// ForkDeps lists the dependencies of the Fork on top of Rest plugin.
type ForkDeps struct {
	// DefaultHTTP is used if there is no different configuration
	DefaultHTTP HTTPHandlers //inject

	Log                 logging.PluginLogger //inject
	PluginName          core.PluginName      //inject
	config.PluginConfig                      //inject
}

// Init checks config if the port is different that it creates ne HTTP server
func (plugin *ForkPlugin) Init() (err error) {
	if plugin.Config == nil {
		plugin.Config = DefaultConfig()
	}
	if err := PluginConfig(plugin.Deps.PluginConfig, plugin.Config, plugin.Deps.PluginName); err != nil {
		return err
	}

	probePort := plugin.Config.GetPort()
	plugin.Deps.Log.WithField("probePort", probePort).Info("init")
	if probePort > 0 && probePort != plugin.Deps.DefaultHTTP.GetPort() {
		childPlugNameHTTP := plugin.String() + "-HTTP"
		plugin.newPlugin = &Plugin{Deps: Deps{
			Log:        logging.ForPlugin(childPlugNameHTTP, plugin.Deps.Log),
			PluginName: core.PluginName(childPlugNameHTTP),
		}, Config: plugin.Config,
		}

		plugin.delegate = plugin.newPlugin
	} else {
		plugin.delegate = plugin.Deps.DefaultHTTP
	}

	if plugin.newPlugin != nil {
		return plugin.newPlugin.Init()
	}

	return err
}

// RegisterHTTPHandler registers HTTP <handler> at the given <path>.
// (delegated call)
func (plugin *ForkPlugin) RegisterHTTPHandler(path string,
	handler func(formatter *render.Render) http.HandlerFunc,
	methods ...string) *mux.Route {

	if plugin.delegate != nil {
		return plugin.delegate.RegisterHTTPHandler(path, handler, methods...)
	}

	plugin.Deps.Log.Warn("not set delegate")
	return nil
}

// GetPort returns plugin configuration port
// (delegated call)
func (plugin *ForkPlugin) GetPort() int {
	if plugin.delegate != nil {
		return plugin.delegate.GetPort()
	}
	return 0
}

// AfterInit starts the HTTP server.
// (only if port was different in Init())
func (plugin *ForkPlugin) AfterInit() error {
	if plugin.newPlugin != nil {
		return plugin.newPlugin.AfterInit()
	}
	return nil
}

// Close stops the HTTP server.
// (only if port was different in Init())
func (plugin *ForkPlugin) Close() error {
	if plugin.newPlugin != nil {
		return plugin.newPlugin.Close()
	}
	return nil
}

// String returns plugin name (if not set defaults to "HTTP-FORK")
func (plugin *ForkPlugin) String() string {
	if plugin.Deps.PluginName != "" {
		return string(plugin.Deps.PluginName)
	}
	return "HTTP-FORK"
}
