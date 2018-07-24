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

// Package core manages the lifecycle of all plugins (start, graceful
// shutdown) and defines the core lifecycle SPI. The core lifecycle SPI
// must be implemented by each plugin.

package deptools

import (
	"fmt"
	"reflect"

	"github.com/ligato/networkservicemesh/plugins/idempotent"
)

const (
	depsFieldname    = "Deps"
	optionalTagName  = "empty_value_ok"
	optionalTagValue = "true"
)

// Check checks to make sure all fields in a Deps are set
// unless the fieldname is in the whitelist, in which case a
// zero value for that field is acceptable
func Check(p idempotent.PluginAPI) error {
	sf, vdeps, err := GetStructFieldValue(p)
	if err != nil {
		return err
	}
	if sf == nil {
		return nil
	}
	return check(sf, vdeps)
}

func check(structField *reflect.StructField, value *reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("CheckDeps requires Deps to be a struct, but it is a %s", value.Kind())
	}
	for i := 0; i < value.NumField(); i++ {
		sf := structField.Type.Field(i)
		if sf.PkgPath != "" {
			return fmt.Errorf("Deps fieldname \"%s\" cannot be lowercase. Deps cannot have private fields.  All fieldnames must be Uppercase", sf.Name)
		}
		v := value.Field(i)
		tag, ok := sf.Tag.Lookup(optionalTagName)
		if (!ok || tag != optionalTagValue) && v.CanInterface() {
			field := v.Interface()
			zeroValue := reflect.Zero(v.Type()).Interface()
			if field == nil || field == zeroValue {
				return fmt.Errorf("field in Deps struct named %s is zero valued, and lacks tag `empty_value_ok:\"true\"`.  Please either set it to a non-zero value or add the `empty_value_ok:\"true\"` to indicate its acceptable for it to be zero valued", sf.Name)
			}
		}
	}
	return nil
}
