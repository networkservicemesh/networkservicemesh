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

const (
	//NSMSocketFile defines socket name which will be used for memif connection
	NSMSocketFile = "socketfile"
	//NSMMaster if true, than role is master
	NSMMaster = "master"
	//NSMSlave if true, than role is slave
	NSMSlave = "slave"
	//NSMPerPodDirectory defines directory that is mounted to pod (relative to /var/lib/networkservicemesh)
	NSMPerPodDirectory = "directory"
)

type MemifInterface struct{}

func (m MemifInterface) ValidateParameters(parameters map[string]string) error {
	keysList := nsmutils.Keys{
		NSMSocketFile:      nsmutils.KeyProperties{Validator: nsmutils.Empty},
		NSMMaster:          nsmutils.KeyProperties{Validator: nsmutils.Bool},
		NSMSlave:           nsmutils.KeyProperties{Validator: nsmutils.Bool},
		NSMPerPodDirectory: nsmutils.KeyProperties{Mandatory: true, Validator: nsmutils.Empty},
	}

	return nsmutils.ValidateParameters(parameters, keysList)
}

func (m MemifInterface) CreateLocalConnect(apiCh govppapi.Channel, src, dst map[string]string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m MemifInterface) DeleteLocalConnect(apiCh govppapi.Channel, connID string) error {
	return nil
}
