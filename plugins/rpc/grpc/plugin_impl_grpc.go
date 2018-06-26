// Copyright (c) 2018 Cisco and/or its affiliates.
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
	"github.com/ligato/cn-infra/rpc/grpc"
	"github.com/ligato/networkservicemesh/utils/idempotent"
)

type Plugin struct {
	idempotent.Impl
	grpc.Plugin
}

func (plugin *Plugin) Init() error {
	return plugin.IdempotentInit(plugin.init)
}

func (plugin *Plugin) init() error {
	err := plugin.Plugin.Init()
	if err != nil {
		return err
	}
	err = idempotent.SafeInit(plugin.Deps.HTTP)
	if err != nil {
		return err
	}
	err = plugin.Plugin.AfterInit()
	return err
}

func (plugin *Plugin) Close() error {
	return plugin.IdempotentClose(plugin.Plugin.Close)
}

func (plugin *Plugin) close() error {
	// Its important that both of plugin and plugin.Deps.HTTP are closed
	err := plugin.IdempotentClose(plugin.close)
	// plugin must be closed before we can close its dependency
	httpErr := idempotent.SafeClose(plugin.Deps.HTTP)
	if err != nil {
		return err
	}
	if httpErr != nil {
		return httpErr
	}
	return nil
}
