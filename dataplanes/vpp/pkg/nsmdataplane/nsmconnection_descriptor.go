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

package nsmdataplane

import (
	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmvpp"
	dataplaneapi "github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ConnectionDescriptor struct {
	src                 map[string]string
	dst                 map[string]string
	crossConnectionType nsmvpp.CrossConnectionType
}

func (c ConnectionDescriptor) Connect(apiCh govppapi.Channel) (string, error) {
	if err := c.crossConnectionType.Validate(c.src, c.dst); err != nil {
		return "", err
	}

	return c.crossConnectionType.Connect(apiCh, c.src, c.dst)
}

func (c ConnectionDescriptor) Disconnect(apiCh govppapi.Channel) error {
	return c.crossConnectionType.Disconnect(apiCh, c.src, c.dst)
}

func BuildConnectionDescriptor(request *dataplaneapi.Connection) (ConnectionDescriptor, error) {
	src := request.LocalSource

	switch request.Destination.(type) {
	case *dataplaneapi.Connection_Local:
		dst := request.Destination.(*dataplaneapi.Connection_Local).Local
		if src.Type == dst.Type {
			return ConnectionDescriptor{
				crossConnectionType: nsmvpp.CreateSameTypeConnection(src.Type),
				src:                 src.Parameters,
				dst:                 dst.Parameters,
			}, nil
		} else {
			return ConnectionDescriptor{
				crossConnectionType: nsmvpp.CreateDifferentTypeConnection(src.Type, dst.Type),
				src:                 src.Parameters,
				dst:                 dst.Parameters,
			}, nil
		}
	case *dataplaneapi.Connection_Remote:
		return ConnectionDescriptor{}, status.Error(codes.Unavailable, "Remote Destination currently is not supported")
	default:
		return ConnectionDescriptor{}, status.Error(codes.Unknown, "Unknown destination type")
	}
}
