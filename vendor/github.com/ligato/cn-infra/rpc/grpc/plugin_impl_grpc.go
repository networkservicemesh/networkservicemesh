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
	"io"

	"net/http"

	"reflect"
	"strconv"

	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/rpc/rest"
	"github.com/ligato/cn-infra/utils/safeclose"
	"github.com/unrolled/render"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

// Plugin maintains the GRPC netListener (see Init, AfterInit, Close methods)
type Plugin struct {
	Deps
	*Config

	// Used mainly for testing purposes
	listenAndServe ListenAndServe

	netListener   io.Closer
	grpcServer    *grpc.Server
	grpcServerVal reflect.Value
}

// FromExisting is a helper for preconfigured GRPC Server
func FromExisting(server *grpc.Server) *Plugin {
	return &Plugin{grpcServer: server}
}

// Deps lists the dependencies of the Rest plugin.
type Deps struct {
	Log        logging.PluginLogger //inject
	PluginName core.PluginName      //inject
	config.PluginConfig             //inject
	HTTP       rest.HTTPHandlers    //inject optional
}

// Init is the plugin entry point called by Agent Core
// - It prepares GRPC netListener for registration of individual service
func (plugin *Plugin) Init() (err error) {
	if plugin.Config == nil {
		plugin.Config = DefaultConfig()
	}
	if err := PluginConfig(plugin.Deps.PluginConfig, plugin.Config, plugin.Deps.PluginName); err != nil {
		return err
	}

	if plugin.grpcServer == nil {
		opts := []grpc.ServerOption{}
		if plugin.Config.MaxConcurrentStreams > 0 {
			opts = append(opts, grpc.MaxConcurrentStreams(plugin.Config.MaxConcurrentStreams))
		}
		if plugin.Config.MaxMsgSize > 0 {
			opts = append(opts, grpc.MaxMsgSize(plugin.Config.MaxMsgSize))
		}
		//TODO plugin.Config: TLS
		//opts = append(opts, grpc.Creds(credentials.NewTLS(nil)))

		plugin.grpcServer = grpc.NewServer(opts...)
		grpclog.SetLogger(plugin.Log.NewLogger("server"))
	}
	plugin.grpcServerVal = reflect.ValueOf(plugin.grpcServer)

	return err
}

// Server is a getter for accessing grpc.Server (of a GRPC plugin)
//
// Example usage:
//
//   protocgenerated.RegisterServiceXY(plugin.Deps.GRPC.Server(), &ServiceXYImplP{})
//
//   type Deps struct {
//       GRPC grps.Server // inject plugin implementing RegisterHandler
//       // other dependencies ...
//   }
func (plugin *Plugin) Server() *grpc.Server {
	return plugin.grpcServer
}

// GetPort returns plugin configuration port
func (plugin *Plugin) GetPort() int {
	if plugin.Config != nil {
		return plugin.Config.GetPort()
	}
	return 0
}

// AfterInit starts the HTTP netListener.
func (plugin *Plugin) AfterInit() (err error) {
	cfgCopy := *plugin.Config

	if plugin.listenAndServe != nil {
		plugin.netListener, err = plugin.listenAndServe(cfgCopy, plugin.grpcServer)
	} else {
		plugin.Log.Info("Listening GRPC on tcp://", cfgCopy.Endpoint)
		plugin.netListener, err = ListenAndServeGRPC(cfgCopy, plugin.grpcServer)
	}

	if plugin.Deps.HTTP != nil {
		plugin.Log.Info("exposing GRPC services over HTTP port " + strconv.Itoa(plugin.Deps.HTTP.GetPort()) +
			" /service ")
		plugin.Deps.HTTP.RegisterHTTPHandler("service", func(formatter *render.Render) http.HandlerFunc {
			return plugin.grpcServer.ServeHTTP
		}, "GET", "PUT", "POST")
	}

	return err
}

// Close stops the HTTP netListener.
func (plugin *Plugin) Close() error {
	wasError := safeclose.Close(plugin.netListener)

	plugin.grpcServer.Stop()

	return wasError
}

// String returns plugin name (if not set defaults to "HTTP")
func (plugin *Plugin) String() string {
	if plugin.Deps.PluginName != "" {
		return string(plugin.Deps.PluginName)
	}
	return "GRPC"
}
