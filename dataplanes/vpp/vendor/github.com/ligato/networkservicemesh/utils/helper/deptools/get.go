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
	"fmt"
	"reflect"

	"github.com/ligato/networkservicemesh/plugins/idempotent"
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

// GetStructFieldValue - Get the StructField and Value of the Deps field in the Plugin p
func GetStructFieldValue(p idempotent.PluginAPI) (*reflect.StructField, *reflect.Value, error) {
	if p == nil {
		return nil, nil, fmt.Errorf("cannot call getDeps on nil")
	}
	t := dereferenceType(reflect.TypeOf(p))
	v := dereferenceValue(reflect.ValueOf(p))
	if t.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("getDeps requires a plugin that is a struct, %s is a %s", t, t.Kind())
	}
	sf, ok := t.FieldByName(depsFieldname)
	if !ok {
		return nil, nil, nil
	}
	vdeps := v.FieldByName(depsFieldname)
	return &sf, &vdeps, nil
}

// Get - Get the Deps from the Plugin
func Get(p idempotent.PluginAPI) (interface{}, error) {
	_, vdeps, err := GetStructFieldValue(p)
	if vdeps != nil {
		return vdeps.Interface(), err
	}
	return nil, err
}
