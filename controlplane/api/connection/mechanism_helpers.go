// Copyright 2018-2019 Cisco Systems, Inc.
// SPDX-License-Identifier: Apache-2.0
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

package connection

import (
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// Equals returns if mechanism equals given mechanism
func (m *Mechanism) Equals(mechanism *Mechanism) bool {
	return proto.Equal(m, mechanism)
}

// Clone clones mechanism
func (m *Mechanism) Clone() *Mechanism {
	return proto.Clone(m).(*Mechanism)
}

var mechanismValidators map[string]func(*Mechanism) error
var mechanismValidatorsMutex sync.Mutex

// AddMechanism adds a Mechanism
func AddMechanism(mtype string, validator func(*Mechanism) error) {
	mechanismValidatorsMutex.Lock()
	defer mechanismValidatorsMutex.Unlock()
	mechanismValidators[mtype] = validator
}

// IsValid - is the Mechanism Valid?
func (m *Mechanism) IsValid() error {
	if m == nil {
		return errors.New("mechanism cannot be nil")
	}
	validator, ok := mechanismValidators[m.GetType()]
	if ok {
		return validator(m)
	}
	// NOTE: this means that we intentionally decide that Mechanisms are valid
	// unless we have a Validator that says otherwise
	return nil
}
