package vppagent

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
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
func (c *Commit) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)
	if vppAgentConfig == nil {
		return nil, fmt.Errorf("received empty VppAgentConfig")
	}

	endpoint.Log(ctx).Infof("Sending VppAgentConfig to VPP Agent: %v", vppAgentConfig)

	if err := c.send(ctx, vppAgentConfig); err != nil {
		return nil, fmt.Errorf("failed to send vppAgentConfig to VPP Agent: %v", err)
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
func (c *Commit) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)

	if vppAgentConfig == nil {
		return nil, fmt.Errorf("received empty vppAgentConfig")
	}

	endpoint.Log(ctx).Infof("Sending vppAgentConfig to VPP Agent: %v", vppAgentConfig)

	if err := c.remove(ctx, vppAgentConfig); err != nil {
		return nil, fmt.Errorf("failed to send DataChange to VPP Agent: %v", err)
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
func (c *Commit) Init(*endpoint.InitContext) error {
	if c.shouldResetVpp {
		return c.init()
	}
	return nil
}

func (c *Commit) createConnection(ctx context.Context) (*grpc.ClientConn, error) {
	start := time.Now()
	logrus.Info("Creating connection to vppagent")
	if err := tools.WaitForPortAvailable(ctx, "tcp", c.Endpoint, createConnectionSleep); err != nil {
		return nil, err
	}

	rv, err := grpc.Dial(c.Endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("can't dial grpc server: %v", err)
	}
	logrus.Infof("Connection to vppagent created.  Elapsed time: %s", time.Since(start))

	return rv, nil
}

func (c *Commit) send(ctx context.Context, dataChange *configurator.Config) error {
	conn, err := c.createConnection(context.TODO())
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.Errorf("error closing dataplane connection %v", err)
		}
	}()
	client := configurator.NewConfiguratorClient(conn)

	if _, err := client.Update(ctx, &configurator.UpdateRequest{Update: dataChange, FullResync: false}); err != nil {
		logrus.Errorf("failed to updsate vpp configuration %v. trying to delete", err)
		if _, deleteErr := client.Delete(ctx, &configurator.DeleteRequest{Delete: dataChange}); deleteErr != nil {
			logrus.Errorf("failed to delete vpp configuration %v", deleteErr)
		}
		return err
	}
	return nil
}

func (c *Commit) remove(ctx context.Context, dataChange *configurator.Config) error {
	conn, err := c.createConnection(context.TODO())
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.Errorf("Failed to close vpp client connection")
		}
	}()
	client := configurator.NewConfiguratorClient(conn)

	if _, err := client.Delete(ctx, &configurator.DeleteRequest{Delete: dataChange}); err != nil {
		return err
	}
	return nil
}

// Reset - Resets vppagent
func (c *Commit) init() error {
	if c.shouldResetVpp {
		if err := c.resetVpp(); err != nil {
			logrus.Errorf("failed to reset vpp agent %v", err)
			return err
		}
	}
	return nil
}

func (c *Commit) resetVpp() error {
	conn, err := c.createConnection(context.TODO())
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			logrus.Errorf("failed to close vpp agent connection %v", err)
		}
	}()
	client := configurator.NewConfiguratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), createConnectionTimeout)
	defer cancel()
	_, err = client.Update(ctx, &configurator.UpdateRequest{
		Update:     &configurator.Config{},
		FullResync: true,
	})
	if err != nil {
		return err
	}
	return nil
}
