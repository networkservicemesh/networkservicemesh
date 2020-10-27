// Copyright (c) 2019 Cisco Systems, Inc.
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

package memif

import (
	"strconv"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
)

type Mechanism interface {
	GetSocketFilename() string
	GetWorkspace() string
	GetNetNsInode() string
	GetMode() (uint32, error)
}

type mechanism struct {
	*connection.Mechanism
}

func ToMechanism(m *connection.Mechanism) Mechanism {
	if m.GetType() == MECHANISM {
		return &mechanism{
			m,
		}
	}
	return nil
}

func (m *mechanism) GetWorkspace() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[common.Workspace]
}

// GetSocketFilename returns memif mechanism socket filename
func (m *mechanism) GetSocketFilename() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[SocketFilename]
}
func (m *mechanism) GetNetNsInode() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[common.NetNsInodeKey]
}

// GetMode returns memif connection mode
func (m *mechanism) GetMode() (uint32, error) {
	if m == nil {
		return 0, errors.New("mechanism cannot be nil")
	}

	if m.GetParameters() == nil {
		return 0, errors.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	modeStr, ok := m.GetParameters()[Mode]
	if !ok {
		return 0, errors.Errorf("mechanism.Type %s requires mechanism.Parameters[%s]", m.GetType(), Mode)
	}

	mode, err := strconv.ParseUint(modeStr, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "mechanism.Parameters[%s] must be a valid", Mode)
	}
	return uint32(mode), nil
}
