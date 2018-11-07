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

package main

import (
	"context"
	"fmt"
	"git.fd.io/govpp.git/extras/libmemif"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
)

func main() {
	// For NSC to program container's dataplane, container's linux namespace must be sent to NSM
	linuxNS, err := tools.GetCurrentNS()
	if err != nil {
		logrus.Fatalf("nsc: failed to get a linux namespace with error: %+v, exiting...", err)
		os.Exit(1)
	}
	logrus.Infof("Starting NSC, linux namespace: %s...", linuxNS)

	var workspace string
	var perPodDirectory string

	if os.Getenv(nsmd.NsmDevicePluginEnv) != "" {
		workspace = nsmd.DefaultWorkspace
		perPodDirectory = os.Getenv(nsmd.NsmPerPodDirectoryEnv)
	} else {
		workspace, err = nsmd.RequestWorkspace()
		if err != nil {
			logrus.Fatalf("nsc: failed set up client connection, error: %+v, exiting...", err)
			os.Exit(1)
		}
		_, perPodDirectory = path.Split(workspace)
	}

	clientSocket := path.Join(workspace, nsmd.ClientSocket)

	logrus.Infof("Connecting to nsm server on socket: %s...", clientSocket)
	if _, err := os.Stat(clientSocket); err != nil {
		logrus.Errorf("nsc: failure to access nsm socket at %s with error: %+v, exiting...", clientSocket, err)
		os.Exit(1)
	}

	conn, err := tools.SocketOperationCheck(clientSocket)
	if err != nil {
		logrus.Fatalf("nsm client: failure to communicate with the socket %s with error: %+v", clientSocket, err)
		os.Exit(1)
	}
	defer conn.Close()

	// Init related activities start here
	nsmConnectionClient := nsmconnect.NewClientConnectionClient(conn)

	_, err = nsmConnectionClient.RequestConnection(context.Background(), &nsmconnect.ConnectionRequest{
		RequestId:          linuxNS,
		LinuxNamespace:     linuxNS,
		NetworkServiceName: "gold-network",
		LocalMechanisms: []*common.LocalMechanism{
			{
				Type: common.LocalMechanismType_KERNEL_INTERFACE,
			},
			{
				Type: common.LocalMechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					nsmutils.NSMSocketFile:      "nsc-memif.sock",
					nsmutils.NSMMaster:          "true",
					nsmutils.NSMPerPodDirectory: perPodDirectory,
				},
			},
		},
	})

	if err != nil {
		logrus.Fatalf("failure to request connection with error: %+v", err)
		os.Exit(1)
	}

	appName := "nsc"
	err = libmemif.Init(appName)
	if err != nil {
		fmt.Printf("libmemif.Init() error: %v\n", err)
		return
	}
	defer libmemif.Cleanup()

	memifCallbacks := &libmemif.MemifCallbacks{
		OnConnect:    memifOnConnect,
		OnDisconnect: memifOnDisconnect,
	}

	memifConfig := &libmemif.MemifConfig{
		MemifMeta: libmemif.MemifMeta{
			IfName:         "memif1",
			SocketFilename: path.Join(workspace, "/memif", "3:nsm-2:3:nsm-1", "nsc-memif.sock"),
			IsMaster:       true,
			Mode:           libmemif.IfModeEthernet,
		},
		MemifShmSpecs: libmemif.MemifShmSpecs{
			NumRxQueues:  3,
			NumTxQueues:  3,
			BufferSize:   2048,
			Log2RingSize: 10,
		},
	}

	logrus.Infof("Callbacks: %+v\n", memifCallbacks)
	logrus.Infof("Config: %+v\n", memifConfig)

	memif, err := libmemif.CreateInterface(memifConfig, memifCallbacks)
	if err != nil {
		fmt.Printf("libmemif.CreateInterface() error: %v\n", err)
		return
	}
	defer memif.Close()

	// Init related activities ends here
	logrus.Info("nsm client: initialization is completed successfully, wait for Ctrl+C...")

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		wg.Done()
	}()
	wg.Wait()
}

func memifOnConnect(memif *libmemif.Memif) error {
	logrus.Info("Connected")
	return nil
}

func memifOnDisconnect(memif *libmemif.Memif) error {
	logrus.Info("Disconnected")
	return nil
}
