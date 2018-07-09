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

package helper

import (
	"fmt"
	"reflect"

	"github.com/ligato/networkservicemesh/plugins/idempotent"
)

const (
	depsFieldname    = "Deps"
	optionalTagName  = "optional"
	optionalTagValue = "true"
)

func dereferenceType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func dereferenceValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// CheckDeps checks to make sure all fields in a Deps are set
// unless the fieldname is in the whitelist, in which case a
// zero value for that field is acceptable
func CheckDeps(p idempotent.PluginAPI) error {
	sf, vdeps, err := getDeps(p)
	if err != nil {
		return err
	}
	if sf == nil {
		return nil
	}
	return checkDeps(sf, vdeps)
}

func getDeps(p idempotent.PluginAPI) (*reflect.StructField, *reflect.Value, error) {
	if p == nil {
		return nil, nil, fmt.Errorf("Cannot call CheckDeps on nil")
	}
	t := dereferenceType(reflect.TypeOf(p))
	v := dereferenceValue(reflect.ValueOf(p))
	if t.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("CheckDeps requires a plugin that is a struct,%s is a %s", t, t.Kind())
	}
	sf, ok := t.FieldByName(depsFieldname)
	if !ok {
		return nil, nil, nil
	}
	vdeps := v.FieldByName(depsFieldname)
	return &sf, &vdeps, nil
}

func checkDeps(structField *reflect.StructField, value *reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("Check%s requires %s to be a struct, but it is a %s", structField.Name, structField.Name, value.Kind())
	}
	for i := 0; i < value.NumField(); i++ {
		sf := structField.Type.Field(i)
		if sf.PkgPath != "" {
			return fmt.Errorf("%s fieldname \"%s\" cannot be lowercase. %s cannot have private fields.  All fieldnames must be Uppercase", structField.Name, structField.Name, sf.Name)
		}
		v := value.Field(i)
		if v.Kind() != reflect.Interface {
			return fmt.Errorf("field in %s struct named %s is not an an interface", structField.Name, sf.Name)
		}
		if !v.IsValid() || v.IsNil() {
			tag, ok := sf.Tag.Lookup(optionalTagName)
			if !ok || tag != optionalTagValue {
				return fmt.Errorf("field in % struct  named %s is zero valued, and lacks tag `optional:\"true\"`.  Please either set it to a non-zero value or add the `optional:\"true\"` to indicate its acceptable for it to be zero valued", structField.Name, sf.Name)
			}
		}
	}
	return nil
}
