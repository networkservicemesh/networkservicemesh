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

package config

import (
	"github.com/ligato/networkservicemesh/plugins/idempotent"
)

// Config struct  this can literally be anything but use mapstruct:
// flag and usage tags to indicate you want to feed it from a flag
type Config interface{}

// Loader load config
type Loader interface {
	LoadConfig() Config
}

// LoaderPlugin is a Plugin that loads configs
type LoaderPlugin interface {
	Loader
	idempotent.PluginAPI
}
