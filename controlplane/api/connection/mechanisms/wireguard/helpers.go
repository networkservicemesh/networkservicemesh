// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
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

package wireguard

import (
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
)

// Mechanism - a wireguard mechanism utility wrapper
type Mechanism interface {
	// SrcIP -  src ip
	SrcIP() (string, error)
	// DstIP - dst ip
	DstIP() (string, error)
	// SrcPublicKey - source public key
	SrcPublicKey() (string, error)
	// DstPublicKey - source public key
	DstPublicKey() (string, error)
}

type mechanism struct {
	*connection.Mechanism
}

// ToMechanism - convert unified mechanism to useful wrapper
func ToMechanism(m *connection.Mechanism) Mechanism {
	if m.Type == MECHANISM {
		return &mechanism{
			m,
		}
	}
	return nil
}

func (m *mechanism) SrcIP() (string, error) {
	return common.GetSrcIP(m.Mechanism)
}

func (m *mechanism) DstIP() (string, error) {
	return common.GetDstIP(m.Mechanism)
}

// SrcPublicKey returns the SrcPublicKey parameter of the Mechanism
func (m *mechanism) SrcPublicKey() (string, error) {
	if m == nil {
		return "", errors.New("mechanism cannot be nil")
	}

	if m.GetParameters() == nil {
		return "", errors.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	srcPublicKey, ok := m.Parameters[SrcPublicKey]
	if !ok {
		return "", errors.Errorf("mechanism.Type %s requires mechanism.Parameters[%s]", m.GetType(), SrcPublicKey)
	}

	return srcPublicKey, nil
}

// DstPublicKey returns the DstPublicKey parameter of the Mechanism
func (m *mechanism) DstPublicKey() (string, error) {
	if m == nil {
		return "", errors.New("mechanism cannot be nil")
	}

	if m.GetParameters() == nil {
		return "", errors.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	dstPublicKey, ok := m.Parameters[DstPublicKey]
	if !ok {
		return "", errors.Errorf("mechanism.Type %s requires mechanism.Parameters[%s]", m.GetType(), DstPublicKey)
	}

	return dstPublicKey, nil
}
