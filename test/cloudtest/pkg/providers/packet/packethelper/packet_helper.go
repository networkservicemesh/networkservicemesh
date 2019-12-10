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

// Package packethelper - A Packet.com helper utils
package packethelper

import (
	"github.com/packethost/packngo"
	"github.com/sirupsen/logrus"
)

// PacketHelper - a heloper utility class for packet.com
type PacketHelper struct {
	// Client - packngo client
	Client *packngo.Client
	// Project - current project
	Project *packngo.Project
	// Projects - a list of all projects available.
	Projects []packngo.Project

	projectID     string
	packetAuthKey string
}

func (ph *PacketHelper) updateProject() error {
	ps, _, err := ph.Client.Projects.List(nil)

	if err != nil {
		logrus.Errorf("Failed to list Packet Projects")
		return err
	}

	ph.Projects = ps

	for i := 0; i < len(ps); i++ {
		p := &ps[i]
		if p.ID == ph.projectID {
			pp := ps[i]
			ph.Project = &pp
		}
	}
	return nil
}

// GetDevices - return list of all devices
func (ph *PacketHelper) GetDevices(id string) (*packngo.Device, *packngo.Response, error) {
	return ph.Client.Devices.Get(id, &packngo.GetOptions{})
}

// NewPacketHelper - construct new helper
func NewPacketHelper(projectID, packetAuthKey string) (*PacketHelper, error) {
	pi := &PacketHelper{
		packetAuthKey: packetAuthKey,
		projectID:     projectID,
	}
	var err error
	if pi.Client = packngo.NewClientWithAuth("cloud-testing-tool", packetAuthKey, nil); pi.Client == nil {
		logrus.Errorf("failed to create Packet REST interface")
		return nil, err
	}

	err = pi.updateProject()
	return pi, err
}
