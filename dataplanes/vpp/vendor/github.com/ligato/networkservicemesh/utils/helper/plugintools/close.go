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
	"github.com/ligato/networkservicemesh/utils/registry"
)

// Close - assist in Init of idempotent.PluginAPI p
// run deptools.Close(p)
// run any provided inits functions
// removes plugin from registry.Shared()
func Close(p idempotent.PluginAPI, closes ...func() error) error {
	return CloseFunc(p, closes...)()
}

// CloseFunc - returns a function that will perform plugintools.Close
func CloseFunc(p idempotent.PluginAPI, closes ...func() error) func() error {
	return func() error {
		err := deptools.Close(p)
		if err != nil {
			return err
		}
		registry.Shared().Delete(p)
		for _, close := range closes {
			err = close()
			if err != nil {
				return err
			}
		}
		return nil
	}
}

// LoggingCloseFunc - returns a function that will perform plugintoos.LoggingClose
func LoggingCloseFunc(log logger.FieldLogger, p idempotent.PluginAPI, inits ...func() error) func() error {
	return func() error {
		start := time.Now()
		if log == nil {
			return fmt.Errorf("argument log to plugintools.LoggingCloseFunc cannot be nil")
		}
		err := Close(p, inits...)
		stop := time.Now()
		duration := stop.Sub(start)
		if err != nil {
			log.Infof("close failed after %v: %s", duration, err)
			return err
		}
		log.Infof("close succeded after %v", duration)
		return err
	}
}
