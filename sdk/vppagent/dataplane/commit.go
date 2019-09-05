package dataplane

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent/dataplane/state"
)

type commit struct {
	*EmptyChainedDataplaneServer
}

func Commit() dataplane.DataplaneServer {
	return &commit{}
}

func (c *commit) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	dataChange, client, err := getDataChangeAndClient(ctx)
	if err != nil {
		return nil, err
	}
	_, err = client.Update(ctx, &configurator.UpdateRequest{Update: dataChange})
	if err != nil {
		return nil, err
	}
	printVppAgentConfiguration(client)
	err = state.CloseConnection(ctx)
	if err != nil {
		return nil, err
	}
	return state.NextDataplaneRequest(ctx, crossConnect)
}

func (c *commit) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	dataChange, client, err := getDataChangeAndClient(ctx)
	if err != nil {
		return nil, err
	}
	_, err = client.Delete(ctx, &configurator.DeleteRequest{Delete: dataChange})
	if err != nil {
		return nil, err
	}
	printVppAgentConfiguration(client)
	err = state.CloseConnection(ctx)
	if err != nil {
		return nil, err
	}
	return state.NextDataplaneClose(ctx, crossConnect)
}

func getDataChangeAndClient(ctx context.Context) (*configurator.Config, configurator.ConfiguratorClient, error) {
	dataChange := state.DataChange(ctx)
	if dataChange == nil {
		return nil, nil, errors.New("dataChange is not passed")
	}
	client := state.ConfigurationClient(ctx)
	if client == nil {
		return nil, nil, errors.New("configuration client is not passed")
	}
	return dataChange, client, nil
}

func printVppAgentConfiguration(client configurator.ConfiguratorClient) {
	dumpResult, err := client.Dump(context.Background(), &configurator.DumpRequest{})
	if err != nil {
		logrus.Errorf("Failed to dump VPP-agent state %v", err)
	}
	logrus.Infof("VPP Agent Configuration: %v", proto.MarshalTextString(dumpResult))
}
