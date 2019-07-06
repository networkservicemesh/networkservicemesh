package vppagent

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	createConnectionTimeout = 120 * time.Second
	createConnectionSleep   = 100 * time.Millisecond
)

// Flush is a VPP Agent Flush composite
type Flush struct {
	Endpoint string
}

type callType int

const (
	isRequest callType = iota + 1
	isClose
)

// Request implements the request handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//	   Next
func (f *Flush) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = WithVppAgentConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := VppAgentConfig(ctx)
	if vppAgentConfig == nil {
		return nil, fmt.Errorf("received empty VppAgentConfig")
	}

	logrus.Infof("Sending VppAgentConfig to VPP Agent: %v", vppAgentConfig)

	if err := f.send(ctx, vppAgentConfig); err != nil {
		logrus.Errorf("Failed to send vppAgentConfig to VPP Agent: %v", err)
		return nil, err
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
func (f *Flush) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	ctx = WithVppAgentConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := VppAgentConfig(ctx)

	if vppAgentConfig == nil {
		return nil, fmt.Errorf("received empty vppAgentConfig")
	}

	logrus.Infof("Sending vppAgentConfig to VPP Agent: %v", vppAgentConfig)

	if err := f.remove(ctx, vppAgentConfig); err != nil {
		logrus.Errorf("Failed to send DataChange to VPP Agent: %v", err)
		return nil, err
	}

	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (f *Flush) Name() string {
	return "flush"
}

// NewFlush creates a Flush
func NewFlush(configuration *common.NSConfiguration, endpoint string) *Flush {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	self := &Flush{
		Endpoint: endpoint,
	}
	return self
}

func (f *Flush) Init(context *endpoint.InitContext) error {
	logrus.Info("Resetting VPP Agent")
	return f.reset()
}

func (f *Flush) createConnection(ctx context.Context) (*grpc.ClientConn, error) {
	if err := tools.WaitForPortAvailable(ctx, "tcp", f.Endpoint, createConnectionSleep); err != nil {
		return nil, err
	}

	tracer := opentracing.GlobalTracer()
	rv, err := grpc.Dial(f.Endpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

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
