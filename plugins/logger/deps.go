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

package logger

import (
	"github.com/ligato/networkservicemesh/plugins/config"
	"github.com/sirupsen/logrus"
	"io"
)

// Deps :
//      Name:         Name of the logger - Added to logger.Plugin.Fields as
//                    {"logname":p.Name}
//      Fields:       logrus.Fields to be added to the logger.Plugin.FieldLogger
//                    All Log messages will carry these fields
//      Hooks:        List of logrus.Hook to be added to logger.Plugin.Log
//      Formatter:    logrus.Formatter to use.
//      Out:          io.Writer to use for outputing Log
//      ConfigLoader: config.LoaderPlugin to load logging config from
//                    config files or other external sources.  ConfigLoader
//                    is only used if no programmatic config is provided.
//                    In the event of no programmatic config and no config
//                    from config files, logger.defaultConfig is used.
type Deps struct {
	Name         string
	Fields       logrus.Fields
	Hooks        []logrus.Hook
	Formatter    logrus.Formatter
	Out          io.Writer
	ConfigLoader config.LoaderPlugin
}
