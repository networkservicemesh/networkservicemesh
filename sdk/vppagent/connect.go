package vppagent

import (
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/ligato/vpp-agent/api/configurator"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

// ConnectionData is a connection data type
type ConnectionData struct {
	InConnName  string
	OutConnName string
	DataChange  *configurator.Config
}

func getConnectionData(endpoint endpoint.ChainedEndpoint, conn *connection.Connection, allowEmpty bool) (*ConnectionData, error) {
	opaque := endpoint.GetOpaque(conn)
	if opaque == nil {
		if allowEmpty {
			return nil, nil
		}
		err := fmt.Errorf("received empty opaque data")
		return nil, err
	}

	connectionData, ok := opaque.(*ConnectionData)
	if !ok {
		err := fmt.Errorf("unexpected opaque data type: expected *vppagent.ConnectionData, received %v", reflect.TypeOf(opaque))
		return nil, err
	}

	dataChangeCopy := proto.Clone(connectionData.DataChange).(*configurator.Config)

	return &ConnectionData{
		InConnName:  connectionData.InConnName,
		OutConnName: connectionData.OutConnName,
		DataChange:  dataChangeCopy,
	}, nil
}
