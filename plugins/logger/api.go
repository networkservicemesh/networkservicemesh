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
	"github.com/go-errors/errors"
	"github.com/ligato/networkservicemesh/plugins/idempotent"
	"github.com/sirupsen/logrus"
)

// FieldLogger is a simple wrapper around logrus.FieldLogger
type FieldLogger interface {
	logrus.FieldLogger
}

type StackLogger interface {
	WithStackTrace(err *errors.Error) logrus.FieldLogger
}

// FieldLoggerPlugin is a FieldLogger and a Plugin
type FieldLoggerPlugin interface {
	idempotent.PluginAPI
	FieldLogger
	StackLogger
}
