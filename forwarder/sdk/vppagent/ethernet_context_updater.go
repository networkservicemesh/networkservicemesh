package forwarder

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

func NewEthernetContextUpdater() forwarder.ForwarderServer {
	return &ethernetContextUpdater{}
}

type ethernetContextUpdater struct {
}

func (c *ethernetContextUpdater) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	updateEthernetContext(ctx, crossConnect)
	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	return next.Request(ctx, crossConnect)
}

func (c *ethernetContextUpdater) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	updateEthernetContext(ctx, crossConnect)
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConnect)
}

func updateEthernetContext(ctx context.Context, c *crossconnect.CrossConnect) {
	getVppDestentaionInterfaceMacByInterfaceName(ctx, c)
	//TODO update ethernet context after problem https://github.com/ligato/vpp-agent/issues/1525 will be solved
}

func getVppDestentaionInterfaceMacByInterfaceName(ctx context.Context, c *crossconnect.CrossConnect) string {
	client := ConfiguratorClient(ctx)
	dumpResp, err := client.Dump(context.Background(), &configurator.DumpRequest{})
	if err != nil {
		Logger(ctx).Errorf("And error during client.Dump: %v", err)
	} else {
		Logger(ctx).Infof("Dump response: %v", dumpResp.String())
	}
	getResp, err := client.Get(ctx, &configurator.GetRequest{})
	if err != nil {
		Logger(ctx).Errorf("And error during client.Get: %v", err)
	} else {
		Logger(ctx).Infof("Get response: %v", getResp.String())
	}
	return ""
}
