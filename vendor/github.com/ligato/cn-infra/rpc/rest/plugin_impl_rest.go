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
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"

	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/utils/safeclose"
)

const (
	// DefaultHTTPPort is used during HTTP server startup unless different port was configured
	DefaultHTTPPort = "9191"
	// DefaultIP 0.0.0.0
	DefaultIP = "0.0.0.0"
	// DefaultEndpoint 0.0.0.0:9191
	DefaultEndpoint = DefaultIP + ":" + DefaultHTTPPort
)

// BasicHTTPAuthenticator is a delegate that implements basic HTTP authentication
type BasicHTTPAuthenticator interface {
	// Authenticate returns true if user is authenticated successfully, false otherwise.
	Authenticate(user string, pass string) bool
}

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
	Log        logging.PluginLogger //inject
	PluginName core.PluginName      //inject
	// Authenticator can be injected in a flavor inject method.
	// If there is no authenticator injected and config contains
	// user password, the default staticAuthenticator is instantiated.
	// By default the authenticator is disabled.
	Authenticator       BasicHTTPAuthenticator //inject
	config.PluginConfig                        //inject
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

	// if there is no injected authenticator and there are credentials defined in the config file
	// instantiate staticAuthenticator otherwise do not use basic Auth
	if plugin.Authenticator == nil && len(plugin.Config.ClientBasicAuth) > 0 {
		plugin.Authenticator, err = newStaticAuthenticator(plugin.Config.ClientBasicAuth)
		if err != nil {
			return err
		}
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

	if plugin.Authenticator != nil {
		return plugin.mx.HandleFunc(path, auth(handler(plugin.formatter), plugin.Authenticator)).Methods(methods...)
	}
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
		if cfgCopy.UseHTTPS() {
			plugin.Log.Info("Listening on https://", cfgCopy.Endpoint)
		} else {
			plugin.Log.Info("Listening on http://", cfgCopy.Endpoint)
		}

		plugin.server, err = ListenAndServeHTTP(cfgCopy, plugin.mx)
	}

	return err
}

// Close stops the HTTP server.
func (plugin *Plugin) Close() error {
	return safeclose.Close(plugin.server)
}

// String returns plugin name (if not set defaults to "HTTP")
func (plugin *Plugin) String() string {
	if plugin.Deps.PluginName != "" {
		return string(plugin.Deps.PluginName)
	}
	return "HTTP"
}

func auth(fn http.HandlerFunc, auth BasicHTTPAuthenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ := r.BasicAuth()
		if !auth.Authenticate(user, pass) {
			w.Header().Set("WWW-Authenticate", "Provide valid username and password")
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}
		fn(w, r)
	}
}

// staticAuthenticator is default implementation of BasicHTTPAuthenticator
type staticAuthenticator struct {
	credentials map[string]string
}

// newStaticAuthenticator creates new instance of static authenticator.
// Argument `users` is a slice of colon-separated username and password couples.
func newStaticAuthenticator(users []string) (*staticAuthenticator, error) {
	sa := &staticAuthenticator{credentials: map[string]string{}}
	for _, u := range users {
		fields := strings.Split(u, ":")
		if len(fields) != 2 {
			return nil, fmt.Errorf("invalid format of basic auth entry '%v' expected 'user:pass'", u)
		}
		sa.credentials[fields[0]] = fields[1]
	}
	return sa, nil
}

// Authenticate looks up the given user name and password in the internal map.
// If match is found returns true, false otherwise.
func (sa *staticAuthenticator) Authenticate(user string, pass string) bool {
	password, found := sa.credentials[user]
	if !found {
		return false
	}
	return pass == password
}
