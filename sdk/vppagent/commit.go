package vppagent

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"google.golang.org/grpc"
)

const (
	createConnectionTimeout = 120 * time.Second
	createConnectionSleep   = 100 * time.Millisecond
)

// Commit is a VPP Agent Commit composite
type Commit struct {
	Endpoint       string
	shouldResetVpp bool
}

// Request implements the request handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//	   Next
func (f *Commit) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)
	if vppAgentConfig == nil {
		return nil, fmt.Errorf("received empty VppAgentConfig")
	}

	endpoint.Log(ctx).Infof("Sending VppAgentConfig to VPP Agent: %v", vppAgentConfig)

	if err := f.send(ctx, vppAgentConfig); err != nil {
		return nil, fmt.Errorf("Failed to send vppAgentConfig to VPP Agent: %v", err)
	}
	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Request(ctx, request)
	}
	return request.GetConnection(), nil
}

// Close implements the close handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//	   Next
func (f *Commit) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)

	if vppAgentConfig == nil {
		return nil, fmt.Errorf("received empty vppAgentConfig")
	}

	endpoint.Log(ctx).Infof("Sending vppAgentConfig to VPP Agent: %v", vppAgentConfig)

	if err := f.remove(ctx, vppAgentConfig); err != nil {
		return nil, fmt.Errorf("Failed to send DataChange to VPP Agent: %v", err)
	}

	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// NewCommit creates a new Commit endpoint.  The Commit endpoint commits
// any changes accumulated in the vppagent.Config in the context.Context
// to vppagent
func NewCommit(configuration *common.NSConfiguration, endpoint string, shouldResetVpp bool) *Commit {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	self := &Commit{
		Endpoint:       endpoint,
		shouldResetVpp: shouldResetVpp,
	}
	return self
}

// Init will reset the vpp shouldResetVpp is true
func (mce *Commit) Init(context *endpoint.InitContext) error {
	if mce.shouldResetVpp {
		return mce.init()
	}
	return nil
}

func (f *Commit) createConnection(ctx context.Context) (*grpc.ClientConn, error) {
	if err := tools.WaitForPortAvailable(ctx, "tcp", f.Endpoint, createConnectionSleep); err != nil {
		return nil, err
	}

	rv, err := tools.DialTCP(f.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("Can't dial grpc server: %v", err)
	}

	return rv, nil
}

func (f *Commit) send(ctx context.Context, dataChange *configurator.Config) error {
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

func (f *Commit) remove(ctx context.Context, dataChange *configurator.Config) error {
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

// Reset - Resets vppagent
func (f *Commit) init() error {
	ctx, cancel := context.WithTimeout(context.Background(), createConnectionTimeout)
	defer cancel()

	conn, err := f.createConnection(ctx)
	if err != nil {
		return nil
	}

	defer func() { _ = conn.Close() }()
	client := configurator.NewConfiguratorClient(conn)
	if f.shouldResetVpp {
		_, err = client.Update(context.Background(), &configurator.UpdateRequest{
			Update:     &configurator.Config{},
			FullResync: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
