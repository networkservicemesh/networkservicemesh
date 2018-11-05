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

package nsmd

import (
	"fmt"
	"strconv"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model/networkservice"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/sirupsen/logrus"
)

func ValidateNetworkServiceRequest(request *networkservice.NetworkServiceRequest) error {
	if request == nil {
		err := fmt.Errorf("NetworkServiceRequest cannot be nil")
		logrus.Error(err)
		return err
	}
	err := ValidateConnection(request.GetConnection(), false)
	if err != nil {
		return err
	}

	localMechanismPreferences := request.GetLocalMechanismPreference()
	// If we don't have preferences, we will default to KERNEL INTERFACE
	if len(localMechanismPreferences) == 0 {
		err := fmt.Errorf("NetworkServiceRequest.LocalMechanismPreferences cannot be zero length")
		logrus.Error(err)
		return err
	}

	for _, localMechanismPreference := range localMechanismPreferences {
		err := ValidateLocalMechanism(localMechanismPreference, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func ValidateConnection(connection *networkservice.Connection, complete bool) error {
	if connection == nil {
		err := fmt.Errorf("NetworkServiceRequest.Connection cannot be nil")
		logrus.Error(err)
		return err
	}
	// TODO check for network services that do not exist
	networkservice := connection.GetNetworkService()
	if networkservice == "" {
		err := fmt.Errorf("NetworkServiceRequest.Connection.NetworkService must not be empty when passed to NetworkService.Request")
		logrus.Error(err)
		return err
	}
	if !complete {
		localMechanism := connection.GetLocalMechanism()
		if localMechanism != nil {
			err := fmt.Errorf("NetworkServiceRequest.Connection.LocalMechanism must be nil when passed to NetworkService.Request")
			logrus.Error(err)
			return err
		}
		connectionID := connection.GetConnectionId()
		if connectionID != "" {
			err := fmt.Errorf("NetworkServiceRequest.Connection.ConnectionId must be empty when passed to NetworkService.Request")
			logrus.Error(err)
			return err
		}
	}
	if complete {
		err := ValidateLocalMechanism(connection.GetLocalMechanism(), complete)
		if err != nil {
			return err
		}
		connectionID := connection.GetConnectionId()
		if connectionID == "" {
			err := fmt.Errorf("Connection.ConnectionId must be not be empty")
			logrus.Error(err)
			return err
		}
	}
	return nil
}

func ValidateLocalMechanism(localMechanism *common.LocalMechanism, complete bool) error {
	if localMechanism == nil {
		err := fmt.Errorf("LocalMechanism must not be nil")
		logrus.Error(err)
		return err
	}
	if localMechanism.GetType() == common.LocalMechanismType_KERNEL_INTERFACE {
		parameters := localMechanism.GetParameters()
		if parameters == nil {
			err := fmt.Errorf("KERNEL_INTERFACE LocalMechanism type requires parameter %s, which is missing", LocalMechanismParameterNetNsInodeKey)
			logrus.Error(err)
			return err
		}
		netnsInode, ok := parameters[LocalMechanismParameterNetNsInodeKey]
		if !ok {
			err := fmt.Errorf("KERNEL_INTERFACE LocalMechanism type requires parameter %s, which is missing", LocalMechanismParameterNetNsInodeKey)
			logrus.Error(err)
			return err
		}
		_, err := strconv.ParseUint(netnsInode, 10, 64)
		if err != nil {
			err := fmt.Errorf("LocalMechanism.Parameters[%s], must be a uint, instead is %s", LocalMechanismParameterNetNsInodeKey, netnsInode)
			logrus.Error(err)
			return err
		}
		if complete {
			_, ok := parameters[LocalMechanismParameterInterfaceNameKey]
			if !ok {
				err := fmt.Errorf("KERNEL_INTERFACE LocalMechanism type requires parameter %s, which is missing", LocalMechanismParameterInterfaceNameKey)
				logrus.Error(err)
				return err
			}
		}
		return nil
	}
	err := fmt.Errorf("Unknown LocalMechanism.Type: %s", localMechanism.GetType())
	logrus.Error(err)
	return err
}
