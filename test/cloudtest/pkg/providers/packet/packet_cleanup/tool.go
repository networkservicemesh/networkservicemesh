// Copyright (c) 2019 Cisco Systems, Inc and/or its affiliates.
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

// Package main - Packet cleanup utils
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers/packet/packethelper"

	"github.com/packethost/packngo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type packetCleanupCmd struct {
	cobra.Command

	cmdArguments *Arguments
}

// Arguments - command line arguments
type Arguments struct {
	clusterPrefix   []string
	sshPrefix       string
	token           string
	projectID       string
	clusterLifetime time.Duration
	deleteSSHKeys   bool
	deleteClusters  bool
}

func initCmd(rootCmd *packetCleanupCmd) {
	rootCmd.Flags().BoolVarP(&rootCmd.cmdArguments.deleteSSHKeys, "ssh-keys", "k", false, "Delete ssh keys")
	rootCmd.Flags().BoolVarP(&rootCmd.cmdArguments.deleteClusters, "clusters", "c", false, "Delete clusters")
	rootCmd.Flags().DurationVarP(&rootCmd.cmdArguments.clusterLifetime, "older", "o", 4, "Cluster usage time in hours, if exceed ssh key/cluster will be deleted")
	rootCmd.Flags().StringVarP(&rootCmd.cmdArguments.token, "token", "t", "", "Packet Token")
	rootCmd.Flags().StringVarP(&rootCmd.cmdArguments.projectID, "project", "p", "", "ProjectId")

	rootCmd.Flags().StringVar(&rootCmd.cmdArguments.sshPrefix, "ssh-prefix", "dev-ci-cloud", "SSH key prefix to delete")
	rootCmd.Flags().StringArrayVar(&rootCmd.cmdArguments.clusterPrefix, "cluster-prefix", []string{"Worker-packet-", "Master-packet-"}, "Cluster name prefix to delete")

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of packet cleanup tool",
		Long:  `All software has versions.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Packet cleanup tool -- HEAD")
		},
	}
	rootCmd.AddCommand(versionCmd)
}

func main() {
	logrus.Infof("Packet cleanup tool.")

	var rootCmd = &packetCleanupCmd{
		cmdArguments: &Arguments{},
	}
	rootCmd.Use = "packet_cleanup"
	rootCmd.Short = "Packet cleanup tool"
	rootCmd.Long = `Cleanup packet instance from ssh keys and old clusters`
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		doCleanup(rootCmd)
	}
	rootCmd.Args = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	initCmd(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func doCleanup(cmd *packetCleanupCmd) {
	if cmd.cmdArguments.projectID == "" || cmd.cmdArguments.token == "" {
		logrus.Errorf("Please specify both projectID and token")
		os.Exit(1)
	}

	helper, err := packethelper.NewPacketHelper(cmd.cmdArguments.projectID, cmd.cmdArguments.token)
	if err != nil {
		logrus.Errorf("Error accessing packet : %v", err)
		os.Exit(1)
	}
	if cmd.cmdArguments.deleteSSHKeys {
		deleteSSHKeys(cmd, helper)
	}
	if cmd.cmdArguments.deleteClusters {
		deleteClusters(cmd, helper)
	}
}

func deleteClusters(cmd *packetCleanupCmd, helper *packethelper.PacketHelper) {
	pageNumber := 0
	for {
		t1 := time.Now()
		devs, _, err := helper.Client.Devices.List(helper.Project.ID, &packngo.ListOptions{
			Page:    pageNumber,
			PerPage: 10,
		})
		if len(devs) == 0 {
			break
		}
		logrus.Infof("Retrieve list of clusters %v", time.Since(t1))
		if err != nil {
			logrus.Errorf("Failed to retrieve list of clusters %v", err)
			break
		}
		for i := 0; i < len(devs); i++ {
			d := &devs[i]
			keyCreateTime, err := time.Parse(time.RFC3339, d.Created)
			if err != nil {
				logrus.Errorf("Failed to parse time %v", err)
				continue
			}
			sinceValue := time.Since(keyCreateTime)
			logrus.Infof("Checking cluster %v uptime: %v, state: %v", d.Hostname, sinceValue, d.State)
			if checkPrefix(d.Hostname, cmd.cmdArguments.clusterPrefix) {
				if sinceValue > cmd.cmdArguments.clusterLifetime*time.Hour {
					logrus.Infof("-----> Cluster %s is marked for deletion.", d.Hostname)
					_, errd := helper.Client.Devices.Delete(d.ID)
					if errd != nil {
						logrus.Errorf("Error during delete %v", errd)
					}
				}
			}
		}
		pageNumber++
	}
	logrus.Infof("All clusters fit into timeframe %v hours", cmd.cmdArguments.clusterLifetime)
}

func checkPrefix(name string, pattern []string) bool {
	for _, p := range pattern {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func deleteSSHKeys(cmd *packetCleanupCmd, helper *packethelper.PacketHelper) {
	sshKeys, _, err := helper.Client.SSHKeys.List()
	if err != nil {
		logrus.Infof("Error retrieve a list of ssh keys: %v", err)
		return
	}
	for i := 0; i < len(sshKeys); i++ {
		k := &sshKeys[i]
		keyCreateTime, err := time.Parse(time.RFC3339, k.Created)
		if err != nil {
			logrus.Infof("Failed to parse time %v", err)
			continue
		}
		sinceTime := time.Since(keyCreateTime)
		logrus.Infof("Checking key %v alive: %v", k.Label, sinceTime)
		if strings.HasPrefix(k.Label, cmd.cmdArguments.sshPrefix) {
			if sinceTime > cmd.cmdArguments.clusterLifetime*time.Hour {
				logrus.Infof("-----> Key marked for deletion")
				_, errd := helper.Client.SSHKeys.Delete(k.ID)
				if errd != nil {
					logrus.Errorf("Error during delete %v", errd)
				}
			}
		}
	}
}
