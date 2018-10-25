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
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
)

type MemifInterface struct{}

func (m MemifInterface) ValidateParameters(parameters map[string]string) error {
	keysList := nsmutils.Keys{
		nsmutils.NSMSocketFile:      nsmutils.KeyProperties{Validator: nsmutils.Empty},
		nsmutils.NSMMaster:          nsmutils.KeyProperties{Validator: nsmutils.Bool},
		nsmutils.NSMSlave:           nsmutils.KeyProperties{Validator: nsmutils.Bool},
		nsmutils.NSMPerPodDirectory: nsmutils.KeyProperties{Mandatory: true, Validator: nsmutils.Empty},
	}

	return nsmutils.ValidateParameters(parameters, keysList)
}

func (m MemifInterface) CreateLocalConnect(apiCh govppapi.Channel, src, dst map[string]string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m MemifInterface) DeleteLocalConnect(apiCh govppapi.Channel, connID string) error {
	return nil
}

func (m MemifInterface) CreateVppInterface(parameters map[string]string) (uint32, error) {
	return 0, nil
}
