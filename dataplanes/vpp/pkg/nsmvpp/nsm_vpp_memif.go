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
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

type parameters map[string]string

const (
	//NSMSocketFile defines socket name which will be used for memif connection
	NSMSocketFile = "socketfile"
	//NSMMaster if true, than role is master
	NSMMaster = "master"
	//NSMSlave if true, than role is slave
	NSMSlave = "slave"
	//NSMPerPodDirectory defines directory that is mounted to pod (relative to /var/lib/networkservicemesh)
	NSMPerPodDirectory = "directory"

	BaseDir = "/var/lib/networkservicemesh/"

	MemifDirectory = "/memif"
)

type MemifInterface struct{}

func (m MemifInterface) ValidateParameters(parameters map[string]string) error {
	keysList := nsmutils.Keys{
		NSMSocketFile:      nsmutils.KeyProperties{Validator: nsmutils.Empty},
		NSMMaster:          nsmutils.KeyProperties{Validator: nsmutils.Bool},
		NSMSlave:           nsmutils.KeyProperties{Validator: nsmutils.Bool},
		NSMPerPodDirectory: nsmutils.KeyProperties{Mandatory: true, Validator: nsmutils.Empty},
	}
	//TODO validate roles

	return nsmutils.ValidateParameters(parameters, keysList)
}

func (m MemifInterface) CreateLocalConnect(apiCh govppapi.Channel, src, dst map[string]string) (string, error) {
	if err := createMemifSocket(src, dst); err != nil {
		return "", err
	}

	return fmt.Sprintf("%v-%v", src[NSMPerPodDirectory], dst[NSMPerPodDirectory]), nil
}

func (m MemifInterface) DeleteLocalConnect(apiCh govppapi.Channel, connID string) error {
	return nil
}

func createMemifSocket(src, dst parameters) error {
	if src[NSMSocketFile] == "" && dst[NSMSocketFile] == "" {
		generatedName := generateSocketName(src, dst)
		src[NSMSocketFile] = generatedName
		dst[NSMSocketFile] = generatedName
	}

	srcSocketDir := buildSocketDir(src)
	dstSocketDir := buildSocketDir(dst)

	if err := os.MkdirAll(srcSocketDir, 0777); err != nil {
		return err
	}

	if err := os.MkdirAll(dstSocketDir, 0777); err != nil {
		return err
	}

	srcSocket := path.Join(srcSocketDir, src[NSMSocketFile])
	dstSocket := path.Join(dstSocketDir, dst[NSMSocketFile])

	if err := os.Symlink(srcSocket, dstSocket); err != nil {
		logrus.Errorf("Fail during creation symlink to socket, because of: %v", err)
	}

	return nil
}

func generateSocketName(src, dst parameters) string {
	return fmt.Sprint("%s-%s.sock", src[NSMPerPodDirectory], dst[NSMPerPodDirectory])
}

func buildSocketDir(p parameters) string {
	return path.Join(BaseDir, p[NSMPerPodDirectory], MemifDirectory)
}
