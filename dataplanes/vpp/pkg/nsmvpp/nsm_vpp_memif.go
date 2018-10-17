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
	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
)

const (
	NSMSocketFile = "socketfile"

	NSMMaster = "master"

	NSMSlave = "slave"
)

var memifKeys = nsmutils.Keys{
	NSMSocketFile: nsmutils.KeyProperties{
		false, func(s string) error { return nil },
	},
	NSMMaster: nsmutils.KeyProperties{
		false, func(s string) error { return nil },
	},
	NSMSlave: nsmutils.KeyProperties{
		false, func(s string) error { return nil },
	},
}

func validateMemifParameters(parameters map[string]string) error {
	return nsmutils.ValidateParameters(parameters, memifKeys)
}

func CreateMemifConnect(apiCh govppapi.Channel, srcParameters, dstParameters map[string]string) (string, error) {
	if err := validateMemifParameters(srcParameters); err != nil {
		return "", err
	}
	if err := validateMemifParameters(dstParameters); err != nil {
		return "", err
	}

	//todo create connection
	return "", nil
}
