// Copyright (c) 2019 Cisco Systems, Inc.
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

package prefix_pool

import (
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	PrefixesFile            = "excluded_prefixes.yaml"
	NsmConfigDir            = "/var/lib/networkservicemesh/config"
	PrefixesFilePathDefault = NsmConfigDir + "/" + PrefixesFile
)

type prefixPoolReader struct {
	prefixPool
	prefixesConfig *viper.Viper
	configPath     string
}

func (ph *prefixPoolReader) init(prefixes []string) {
	ph.prefixes = prefixes
	ph.basePrefixes = prefixes
	ph.connections = map[string]*connectionRecord{}
}

func NewPrefixPoolReader(path string) PrefixPool {
	ph := &prefixPoolReader{
		prefixesConfig: viper.New(),
		configPath:     path,
	}

	// setup watching the prefixes config file
	ph.prefixesConfig.SetDefault("prefixes", []string{})
	ph.prefixesConfig.SetConfigFile(ph.configPath)
	ph.prefixesConfig.ReadInConfig()

	readPrefixes := func() {
		logrus.Infof("Reading excluded prefixes config file: %s", ph.configPath)
		prefixes := ph.prefixesConfig.GetStringSlice("prefixes")
		logrus.Infof("Excluded prefixes: %v", prefixes)
		ph.Lock()
		defer ph.Unlock()
		ph.init(prefixes)
	}

	ph.prefixesConfig.OnConfigChange(func(fsnotify.Event) {
		logrus.Info("Excluded prefixes config file changed")
		readPrefixes()
	})
	ph.prefixesConfig.WatchConfig()
	readPrefixes()

	return ph
}
