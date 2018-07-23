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

package plugintools

import (
	"fmt"
	"time"

	"github.com/ligato/networkservicemesh/plugins/idempotent"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/utils/helper/deptools"
)

// Init - assist in Init of idempotent.PluginAPI p
// run deptools.Init(p)
// run any provided inits functions
// log elapsed time and succes or failure
func Init(p idempotent.PluginAPI, inits ...func() error) error {
	return InitFunc(p, inits...)()
}

// InitFunc - returns a function that will perform plugintools.Init
func InitFunc(p idempotent.PluginAPI, inits ...func() error) func() error {
	return func() error {
		err := deptools.Check(p)
		if err != nil {
			return err
		}
		err = deptools.Init(p)
		if err != nil {
			return err
		}
		for _, init := range inits {
			err = init()
			if err != nil {
				return err
			}
		}
		return nil
	}
}

// LoggingInitFunc - returns a function that will perform plugintoos.LoggingInit
func LoggingInitFunc(log logger.FieldLogger, p idempotent.PluginAPI, inits ...func() error) func() error {
	return func() error {
		start := time.Now()
		if log == nil {
			return fmt.Errorf("argument log to plugintools.LoggingInitFunc cannot be nil")
		}
		err := Init(p, inits...)
		stop := time.Now()
		duration := stop.Sub(start)
		if err != nil {
			log.Infof("initialization failed after %v: %s", duration, err)
			return err
		}
		log.Infof("initialization succeeded after %v", duration)
		return err
	}
}
