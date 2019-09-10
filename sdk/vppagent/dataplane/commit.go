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
)

type commit struct {
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
	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	return next.Request(ctx, crossConnect)
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
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConnect)
}

func getDataChangeAndClient(ctx context.Context) (*configurator.Config, configurator.ConfiguratorClient, error) {
	dataChange := DataChange(ctx)
	if dataChange == nil {
		return nil, nil, errors.New("dataChange is not passed")
	}
	client := ConfigurationClient(ctx)
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

func Commit() dataplane.DataplaneServer {
	return &commit{}
}
