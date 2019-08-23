package vppagent

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

const (
	createConnectionTimeout = 120 * time.Second
	createConnectionSleep   = 100 * time.Millisecond
)

// Flush is a VPP Agent Flush composite
type Flush struct {
	endpoint.BaseCompositeEndpoint
	Endpoint string
}

// Request implements the request handler
func (f *Flush) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if f.GetNext() == nil {
		err := fmt.Errorf("composite requires that there is Next set")
		return nil, err
	}

	incomingConnection, err := f.GetNext().Request(ctx, request)
	if err != nil {
		return nil, err
	}

	connectionData, err := getConnectionData(f.GetNext(), incomingConnection, false)
	if err != nil {
		return nil, err
	}

	dataChange := connectionData.DataChange
	if dataChange == nil {
		err = fmt.Errorf("received empty DataChange")
		return nil, err
	}

	logrus.Infof("Sending DataChange to VPP Agent: %v", dataChange)
	err = f.send(ctx, dataChange)
	if err != nil {
		logrus.Errorf("Failed to send DataChange to VPP Agent: %v", err)
		return nil, err
	}

	return incomingConnection, nil
}

// Close implements the close handler
func (f *Flush) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	connectionData, err := getConnectionData(f.GetNext(), connection, false)
	if err != nil {
		return &empty.Empty{}, err
	}

	dataChange := connectionData.DataChange
	if dataChange == nil {
		err = fmt.Errorf("received empty DataChange")
		return &empty.Empty{}, err
	}

	logrus.Infof("Removing DataChange from VPP Agent: %v", dataChange)
	err = f.remove(ctx, dataChange)
	if err != nil {
		logrus.Errorf("Failed to remove DataChange from VPP Agent: %v", err)
		return &empty.Empty{}, err
	}

	if f.GetNext() != nil {
		return f.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (f *Flush) Name() string {
	return "flush"
}

// NewFlush creates a Flush
func NewFlush(configuration *common.NSConfiguration, endpoint string) *Flush {

	self := &Flush{
		Endpoint: endpoint,
	}

	logrus.Info("Resetting VPP Agent")
	err := self.reset()
	if err != nil {
		logrus.Errorf("Failed to reset VPP Agent: %v", err)
		return nil
	}

	return self
}

func (f *Flush) createConnection(ctx context.Context) (*grpc.ClientConn, error) {
	if err := tools.WaitForPortAvailable(ctx, "tcp", f.Endpoint, createConnectionSleep); err != nil {
		return nil, err
	}

	rv, err := tools.DialTCP(f.Endpoint)
	if err != nil {
		logrus.Errorf("Can't dial grpc server: %v", err)
		return nil, err
	}

	return rv, nil
}

func (f *Flush) send(ctx context.Context, dataChange *configurator.Config) error {
	conn, err := f.createConnection(ctx)
	if err != nil {
		return nil
	}

	defer func() { _ = conn.Close() }()
	client := configurator.NewConfiguratorClient(conn)

	if _, err := client.Update(ctx, &configurator.UpdateRequest{Update: dataChange}); err != nil {
		_, _ = client.Delete(ctx, &configurator.DeleteRequest{Delete: dataChange})
		return err
	}
	return nil
}

func (f *Flush) remove(ctx context.Context, dataChange *configurator.Config) error {
	conn, err := f.createConnection(ctx)
	if err != nil {
		return nil
	}

	defer func() { _ = conn.Close() }()
	client := configurator.NewConfiguratorClient(conn)

	if _, err := client.Delete(ctx, &configurator.DeleteRequest{Delete: dataChange}); err != nil {
		return err
	}
	return nil
}

func (f *Flush) reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), createConnectionTimeout)
	defer cancel()

	conn, err := f.createConnection(ctx)
	if err != nil {
		return nil
	}

	defer func() { _ = conn.Close() }()
	client := configurator.NewConfiguratorClient(conn)

	_, err = client.Update(context.Background(), &configurator.UpdateRequest{
		Update:     &configurator.Config{},
		FullResync: true,
	})
	if err != nil {
		return err
	}
	return nil
}
