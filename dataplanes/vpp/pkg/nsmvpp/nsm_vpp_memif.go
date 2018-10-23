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
	"github.com/ligato/networkservicemesh/dataplanes/vpp/bin_api/memif"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"strconv"
)

type parameters map[string]string

const (
	NSMSocketFile = "socketfile"

	NSMMaster = "master"

	NSMSlave = "slave"

	NSMPerPodDirectory = "directory"

	BaseDir = "/var/lib/networkservicemesh/"
)

var memifKeys = nsmutils.Keys{
	NSMSocketFile: nsmutils.KeyProperties{
		Validator: nsmutils.Empty,
	},
	NSMMaster: nsmutils.KeyProperties{
		Validator: nsmutils.Empty,
	},
	NSMSlave: nsmutils.KeyProperties{
		Validator: nsmutils.Empty,
	},
	NSMPerPodDirectory: nsmutils.KeyProperties{
		Validator: nsmutils.Empty,
	},
}

func validateMemifParameters(parameters map[string]string) error {
	return nsmutils.ValidateParameters(parameters, memifKeys)
}

func CreateMemifConnect(apiCh govppapi.Channel, src, dst parameters) (string, error) {
	if err := validateMemifParameters(src); err != nil {
		return "", err
	}

	if err := validateMemifParameters(dst); err != nil {
		return "", err
	}

	var socketId uint32 = 17
	if err := createMemifSocket(apiCh, src, dst, socketId); err != nil {
		return "", err
	}
	logrus.Info("Memif socket successfully created")

	srcIfIndex, err := createMemifInterface(apiCh, src, 111, socketId)
	if err != nil {
		return "", err
	}
	logrus.Info("Source interface successfully created")

	dstIfIndex, err := createMemifInterface(apiCh, dst, 222, socketId)
	if err != nil {
		return "", err
	}
	logrus.Info("Destination interface successfully created")

	return fmt.Sprintf("%v-%v", srcIfIndex, dstIfIndex), nil
}

func createMemifSocket(apiCh govppapi.Channel, src, dst parameters, socketId uint32) error {
	srcSocketDir := path.Join(BaseDir, src[NSMPerPodDirectory], "memif")
	dstSocketDir := path.Join(BaseDir, dst[NSMPerPodDirectory], "memif")

	if err := os.MkdirAll(srcSocketDir, 0777); err != nil {
		return err
	}

	if err := os.MkdirAll(dstSocketDir, 0777); err != nil {
		return err
	}

	srcSocket := path.Join(srcSocketDir, src[NSMSocketFile])
	dstSocket := path.Join(dstSocketDir, dst[NSMSocketFile])

	socketCreateRequest := &memif.MemifSocketFilenameAddDel{
		IsAdd:          1,
		SocketID:       socketId,
		SocketFilename: []byte(srcSocket),
	}
	socketCreateReply := memif.NewMemifSocketFilenameAddDelReply()
	if err := apiCh.SendRequest(socketCreateRequest).ReceiveReply(socketCreateReply); err != nil {
		return err
	}

	if err := os.Link(srcSocket, dstSocket); err != nil {
		logrus.Errorf("Fail during creation hardlink to socket, because of: %v", err)
	}

	return nil
}

func createMemifInterface(apiCh govppapi.Channel, p parameters, id uint32, socketId uint32) (uint32, error) {
	var role uint8 = 0
	if isMaster, _ := strconv.ParseBool(p[NSMMaster]); isMaster {
		role = 1
	}

	memifCreate := &memif.MemifCreate{
		Role:     role,
		ID:       id,
		SocketID: socketId,
	}

	memifCreateReply := &memif.MemifCreateReply{}
	if err := apiCh.SendRequest(memifCreate).ReceiveReply(memifCreateReply); err != nil {
		return 0, err
	}
	return memifCreateReply.SwIfIndex, nil
}
