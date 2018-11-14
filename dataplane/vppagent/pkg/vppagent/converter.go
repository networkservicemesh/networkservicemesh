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

package vppagent

import (
	"fmt"

	"github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"github.com/sirupsen/logrus"
)

type Converter interface {
	Name() string
	Validate() error
	FullySpecify() error
	ToDataRequest(*rpc.DataRequest) (*rpc.DataRequest, error)
}

type ConnectionConverterFactory func(*dataplane.CrossConnect) Converter
type MechanismConverterFactory func(*dataplane.CrossConnect, SrcDst) Converter

var connectionConverterFactories = []ConnectionConverterFactory{
	// TODO: MemifDirectConnectionConverter,
	NewDefaultConnectionConverter,
}

var mechanismConverterFactories = []MechanismConverterFactory{
	NewKernelInterfaceConverter,
}

func NewConnectionConverter(c *dataplane.CrossConnect) Converter {
	for _, converterFactory := range connectionConverterFactories {
		converter := converterFactory(c)
		logrus.Infof("Attempting Connection Converter: %s", converter.Name())
		err := converter.Validate()
		if err == nil {
			return converter
		}
		logrus.Infof("Failed with Connection Converter: %s: %s", converter.Name(), err)
	}
	return nil
}

func NewMechanismConverter(c *dataplane.CrossConnect, s SrcDst) Converter {
	for _, converterFactory := range mechanismConverterFactories {
		converter := converterFactory(c, s)
		logrus.Infof("Attempting Mechanism Converter: %s for side %s of crossconnect %v", converter.Name(), s, c)
		err := converter.Validate()
		if err == nil {
			return converter
		}
		logrus.Infof("Failed with Mechanism Converter: %s: %s", converter.Name(), err)
	}
	return nil
}

func DataRequestFromConnection(c *dataplane.CrossConnect, d *rpc.DataRequest) (*rpc.DataRequest, error) {
	converter := NewConnectionConverter(c)
	if converter != nil {
		rv, err := converter.ToDataRequest(d)
		if err != nil {
			return nil, err
		}
		return rv, nil
	}
	return nil, fmt.Errorf("Connection unsupported: %+v", c)
}
