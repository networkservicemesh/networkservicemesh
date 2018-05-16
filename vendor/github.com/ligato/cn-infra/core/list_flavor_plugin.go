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

package core

import (
	"errors"
	"os"
	"reflect"

	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/logrus"
)

var flavorLogger = logrus.NewLogger("flavors")

func init() {
	if os.Getenv("DEBUG_FLAVORS") != "" {
		flavorLogger.SetLevel(logging.DebugLevel)
	}
}

// Flavor is a structure that contains a particular combination of plugins
// (fields of plugins).
type Flavor interface {
	// Plugins returns a list of plugins.
	// Name of the plugin is supposed to be related to field name of Flavor struct.
	Plugins() []*NamedPlugin

	// Inject method is supposed to be implemented by each Flavor
	// to inject dependencies between the plugins.
	Injector

	// LogRegistry is a getter for accessing log registry (that allows to create new loggers)
	LogRegistry() logging.Registry
}

// Injector is simple interface reused at least on two places:
// - Flavor
// - NewAgent constructor WithPlugins() option
type Injector interface {
	// When this method is called for the first time it returns true
	// (meaning the dependency injection ran at the first time).
	// It is possible to call this method repeatedly (then it will return false).
	Inject() (firstRun bool)
}

// ListPluginsInFlavor lists plugins in a Flavor.
// It extracts all plugins and returns them as a slice of NamedPlugins.
func ListPluginsInFlavor(flavor Flavor) (plugins []*NamedPlugin) {
	uniqueness := map[Plugin] /*nil*/ interface{}{}
	l, err := listPluginsInFlavor(reflect.ValueOf(flavor), uniqueness)
	if err != nil {
		flavorLogger.Error("Invalid argument - it does not satisfy the Flavor interface")
	}
	return l
}

// listPluginsInFlavor lists all plugins in a Flavor. A Flavor is composed
// of one or more Plugins and (optionally) multiple Inject. The composition
// is recursive: a component Flavor contains Plugin components and may
// contain Flavor components as well. The function recursively lists
// plugins included in component Inject.
//
// The function returns an error if the flavorValue argument does not
// satisfy the Flavor interface. All components in the argument flavorValue
// must satisfy either the Plugin or the Flavor interface. If they do not,
// an error is logged, but the function does not return an error.
func listPluginsInFlavor(flavorValue reflect.Value, uniqueness map[Plugin] /*nil*/ interface{}) ([]*NamedPlugin, error) {
	flavorLogger.Debug("inspect flavor structure ", flavorValue.Type())

	var res []*NamedPlugin

	flavorType := flavorValue.Type()

	if flavorType.Kind() == reflect.Ptr {
		flavorType = flavorType.Elem()
	}

	if flavorValue.Kind() == reflect.Ptr {
		flavorValue = flavorValue.Elem()
	}

	if !flavorValue.IsValid() {
		return res, nil
	}

	if _, ok := flavorValue.Addr().Interface().(Flavor); !ok {
		return res, errors.New("does not satisfy the Flavor interface")
	}

	pluginType := reflect.TypeOf((*Plugin)(nil)).Elem()

	if flavorType.Kind() == reflect.Struct {
		numField := flavorType.NumField()
		for i := 0; i < numField; i++ {
			field := flavorType.Field(i)

			exported := field.PkgPath == "" // PkgPath is empty for exported fields
			if !exported {
				continue
			}

			fieldVal := flavorValue.Field(i)
			plug, implementsPlugin := fieldPlugin(field, fieldVal, pluginType)
			if implementsPlugin {
				if plug != nil {
					_, found := uniqueness[plug]
					if !found {
						uniqueness[plug] = nil
						res = append(res, &NamedPlugin{PluginName: PluginName(field.Name), Plugin: plug})

						flavorLogger.
							WithField("fieldName", field.Name).
							Debug("Found plugin in flavor ", field.Type)
					} else {
						flavorLogger.
							WithField("fieldName", field.Name).
							Debug("Found plugin in flavor with non unique name")
					}
				} else {
					flavorLogger.
						WithField("fieldName", field.Name).
						Debug("Found nil plugin in flavor")
				}
			} else {
				// try to inspect flavor structure recursively
				l, err := listPluginsInFlavor(fieldVal, uniqueness)
				if err != nil {
					flavorLogger.
						WithField("fieldName", field.Name).
						Error("Bad field: must satisfy either Plugin or Flavor interface")
				} else {
					res = append(res, l...)
				}
			}
		}
	}

	return res, nil
}

// fieldPlugin determines if a given field satisfies the Plugin interface.
// If yes, the plugin value is returned; if not, nil is returned.
func fieldPlugin(field reflect.StructField, fieldVal reflect.Value, pluginType reflect.Type) (
	plugin Plugin, implementsPlugin bool) {

	switch fieldVal.Kind() {
	case reflect.Struct:
		ptrType := reflect.PtrTo(fieldVal.Type())
		if ptrType.Implements(pluginType) {
			if fieldVal.CanAddr() {
				if plug, ok := fieldVal.Addr().Interface().(Plugin); ok {
					return plug, true
				}
			}
			return nil, true
		}
	case reflect.Ptr, reflect.Interface:
		if plug, ok := fieldVal.Interface().(Plugin); ok {
			if fieldVal.IsNil() {
				flavorLogger.WithField("fieldName", field.Name).
					Debug("Field is nil ", pluginType)
				return nil, true
			}
			return plug, true
		}

	}
	return nil, false
}

// Inject is a utility if you need to combine multiple flavorAggregator for in first parameter of NewAgent()
// It calls Inject() on every plugin.
//
// Example:
//
//   NewAgent(Inject(&Flavor1{}, &Flavor2{}))
//
func Inject(fs ...Flavor) Flavor {
	ret := flavorAggregator{fs}
	ret.Inject()
	return ret
}

type flavorAggregator struct {
	fs []Flavor
}

// Plugins returns list of plugins af all flavorAggregator
func (flavors flavorAggregator) Plugins() []*NamedPlugin {
	var ret []*NamedPlugin
	for _, f := range flavors.fs {
		ret = appendDiff(ret, f.Plugins()...)
	}
	return ret
}

// Inject returns true if at least one returned true
func (flavors flavorAggregator) Inject() (firstRun bool) {
	ret := false
	for _, f := range flavors.fs {
		ret = ret || f.Inject()
	}
	return ret
}

// LogRegistry is a getter for accessing log registry of first flavor
func (flavors flavorAggregator) LogRegistry() logging.Registry {
	if len(flavors.fs) > 0 {
		flavors.fs[0].LogRegistry()
	}

	return nil
}

// Do not append plugins contained in multiple flavors
func appendDiff(existing []*NamedPlugin, new ...*NamedPlugin) []*NamedPlugin {
	for _, newPlugin := range new {
		exists := false
		for _, existingPlugin := range existing {
			if newPlugin.PluginName == existingPlugin.PluginName {
				flavorLogger.Debugf("duplicate of plugin skipped %v", newPlugin.PluginName)
				exists = true
				break
			}
		}
		if !exists {
			existing = append(existing, newPlugin)
		}
	}
	return existing
}
