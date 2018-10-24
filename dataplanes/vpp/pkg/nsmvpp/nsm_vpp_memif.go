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
	"github.com/docker/docker/pkg/mount"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/sirupsen/logrus"
	"net"
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
	connectionId := buildConnectionId(src, dst)

	if src[NSMSocketFile] == "" && dst[NSMSocketFile] == "" {
		generatedName := connectionId + ".sock"
		src[NSMSocketFile] = generatedName
		dst[NSMSocketFile] = generatedName
	}

	srcSocketDir, err := createSocketFolder(src, connectionId)
	if err != nil {
		return err
	}

	dstSocketDir, err := createSocketFolder(dst, connectionId)
	if err != nil {
		return err
	}

	if err := mount.Mount(srcSocketDir, dstSocketDir, "hard", "bind"); err != nil {
		return err
	}
	logrus.Infof("Successfully mount folder %s to %s", connectionId, dstSocketDir)

	socket := path.Join(srcSocketDir, src[NSMSocketFile])
	//dstSocket := path.Join(dstSocketDir, dst[NSMSocketFile])

	if _, err := net.Listen("unix", socket); err != nil {
		return err
	}

	logrus.Info("Start listening socket: %s", socket)

	return nil
}

func buildConnectionId(src, dst parameters) string {
	return fmt.Sprintf("%s-%s", src[NSMPerPodDirectory], dst[NSMPerPodDirectory])
}

func createSocketFolder(p parameters, connectionId string) (string, error) {
	socketDir := path.Join(BaseDir, p[NSMPerPodDirectory], MemifDirectory, connectionId)

	if err := os.MkdirAll(socketDir, 0777); err != nil {
		return "", err
	}
	logrus.Infof("Create folder for socket: %s", socketDir)

	return socketDir, nil
}
