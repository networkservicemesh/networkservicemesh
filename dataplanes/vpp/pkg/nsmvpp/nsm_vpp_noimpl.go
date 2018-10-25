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

package nsmvpp

import (
	"fmt"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
)

type UnimplementedMechanism struct {
	Type common.LocalMechanismType
}

// CreateLocalConnect return error for unimplemented mechanism
func (m UnimplementedMechanism) CreateLocalConnect(apiCh govppapi.Channel, srcParameters, dstParameters map[string]string) (string, error) {
	return "", fmt.Errorf("%s mechanism not implemented", common.LocalMechanismType_name[int32(m.Type)])
}

func (m UnimplementedMechanism) DeleteLocalConnect(apiCh govppapi.Channel, connID string) error {
	return fmt.Errorf("%s mechanism not implemented", common.LocalMechanismType_name[int32(m.Type)])
}

func (m UnimplementedMechanism) ValidateParameters(parameters map[string]string) error {
	return nil
}

func (m UnimplementedMechanism) CreateVppInterface(parameters map[string]string) (uint32, error) {
	return 0, nil
}
