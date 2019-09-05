package dataplane

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/converter"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent/dataplane/state"
)

type clearMechanism struct {
	*EmptyChainedDataplaneServer
	monitor monitor_crossconnect.MonitorServer
	baseDir string
}

func ClearMechanism(monitor monitor_crossconnect.MonitorServer, baseDir string) dataplane.DataplaneServer {
	return &clearMechanism{
		EmptyChainedDataplaneServer: new(EmptyChainedDataplaneServer),
		monitor:                     monitor,
		baseDir:                     baseDir,
	}
}

func (c *clearMechanism) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	conversionParameters := &converter.CrossConnectConversionParameters{
		BaseDir: c.baseDir,
	}
	entity := c.monitor.Entities()[crossConnect.GetId()]
	if entity != nil {
		clearDataChange, cErr := converter.NewCrossConnectConverter(entity.(*crossconnect.CrossConnect), conversionParameters).MechanismsToDataRequest(nil, false)
		if cErr == nil && clearDataChange != nil {
			logrus.Infof("Sending clearing DataChange to vppagent: %v", proto.MarshalTextString(clearDataChange))
			client := state.ConfigurationClient(ctx)
			if client == nil {
				return nil, errors.New("configuration client is not passed for clear mechanism")
			}
			_, cErr = client.Delete(ctx, &configurator.DeleteRequest{Delete: clearDataChange})
		}
		if cErr != nil {
			logrus.Warnf("Connection Mechanism was not cleared properly before updating: %s", cErr.Error())
		}
	}
	return state.NextDataplaneRequest(ctx, crossConnect)
}
