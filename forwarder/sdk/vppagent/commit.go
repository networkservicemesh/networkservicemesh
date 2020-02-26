package vppagent

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"go.ligato.io/vpp-agent/v3/proto/ligato/configurator"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

type commit struct {
	downstreamResync func()
}

func (c *commit) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	dataChange, client, err := getDataChangeAndClient(ctx)
	if err != nil {
		return nil, err
	}
	updateSpan := spanhelper.FromContext(ctx, "VppAgent.UpdateRequest")
	updateSpan.LogObject("dataChange", dataChange)
	_, err = client.Update(updateSpan.Context(), &configurator.UpdateRequest{Update: dataChange})
	updateSpan.LogError(err)
	if err != nil {
		// Error in vpp-agent Update request may cause vpp - vpp agent desynchronization
		c.downstreamResync()
		return nil, err
	}
	updateSpan.Finish()
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
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConnect)
}

func getDataChangeAndClient(ctx context.Context) (*configurator.Config, configurator.ConfiguratorServiceClient, error) {
	dataChange := DataChange(ctx)
	if dataChange == nil {
		return nil, nil, errors.New("dataChange is not passed")
	}
	client := ConfiguratorClient(ctx)
	if client == nil {
		return nil, nil, errors.New("configuration client is not passed")
	}
	return dataChange, client, nil
}

// Commit creates handler for commits changes to vpp-agent
func Commit(downstreamResync func()) forwarder.ForwarderServer {
	return &commit{
		downstreamResync: downstreamResync,
	}
}
