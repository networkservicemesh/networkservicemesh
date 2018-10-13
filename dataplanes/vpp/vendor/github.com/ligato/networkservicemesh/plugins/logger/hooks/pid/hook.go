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

package pid

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Hook - Logrus Hook for adding current pid to logs as a field
// Note: Doing as a hook because if the process forks, this will change
// So we can't simply set it once at the beginning
type Hook struct {
	Field  string
	levels []logrus.Level
}

// Levels to which to add pid.Hook
func (hook *Hook) Levels() []logrus.Level {
	return hook.levels
}

// Fire when logging and logrus.Entry to which this Hook applies
func (hook *Hook) Fire(entry *logrus.Entry) error {
	entry.Data[hook.Field] = os.Getpid()
	return nil
}

// NewHook - Create a new pid.Hook
func NewHook(levels ...logrus.Level) *Hook {
	hook := Hook{
		Field:  "pid",
		levels: levels,
	}
	if len(hook.levels) == 0 {
		hook.levels = logrus.AllLevels
	}

	return &hook
}
