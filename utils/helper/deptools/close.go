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

package deptools

import (
	"reflect"

	"github.com/ligato/networkservicemesh/plugins/idempotent"
)

// Close closes the Deps of a Plugin
func Close(p idempotent.PluginAPI) error {
	err := Check(p)
	if err != nil {
		return err
	}
	// Note: Any error from getDeps would have been hit previously
	// by CheckDeps
	sf, vdeps, _ := GetStructFieldValue(p)
	if sf == nil {
		return nil
	}
	return close(sf, vdeps)
}

func close(structField *reflect.StructField, value *reflect.Value) error {
	// Deps passed in here is presumed to have passed CheckDeps, so there are many
	// errors we don't need to check for
	// Example: We know Deps is a struct with only Interface fields
	for i := 0; i < value.NumField(); i++ {
		v := value.Field(i)
		if v.CanInterface() {
			p, ok := v.Interface().(idempotent.PluginAPI)
			if ok {
				err := p.Close()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
