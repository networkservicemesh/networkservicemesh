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

// Flush is a VPP Agent Flush composite
type Flush struct {
	endpoint.BaseCompositeEndpoint
	Endpoint string
}

// Request implements the request handler
func (f *Flush) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if f.GetNext() == nil {
		logrus.Fatal("The VPP Agent Flush composite requires that there is Next set")
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
	}
	if err := tools.WaitForPortAvailable(ctx, "tcp", f.Endpoint, 100*time.Millisecond); err != nil {
		return nil, err
	}

	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(f.Endpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	if err != nil {
		logrus.Errorf("Can't dial grpc server: %v", err)
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	client := configurator.NewConfiguratorClient(conn)

	incomingConnection, err := f.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	opaque := f.GetNext().GetOpaque(incomingConnection)
	if opaque == nil {
		err := fmt.Errorf("received empty data from Next")
		logrus.Errorf("Unable to find the DataChange: %v", err)
		return nil, err
	}
	dataChange := opaque.(*ConnectionData).DataChange

	logrus.Infof("Sending DataChange to VPP Agent: %v", dataChange)
	if _, err := client.Update(ctx, &configurator.UpdateRequest{Update: dataChange}); err != nil {
		logrus.Error(err)
		_, _ = client.Delete(ctx, &configurator.DeleteRequest{Delete: dataChange})
		return nil, err
	}

	return incomingConnection, nil
}

func (f *Flush) reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := tools.WaitForPortAvailable(ctx, "tcp", f.Endpoint, 100*time.Millisecond); err != nil {
		return err
	}
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(f.Endpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	if err != nil {
		logrus.Errorf("Can't dial grpc server: %v", err)
		return err
	}
	defer func() { _ = conn.Close() }()
	client := configurator.NewConfiguratorClient(conn)
	logrus.Infof("Resetting VPP Agent...")
	_, err = client.Update(context.Background(), &configurator.UpdateRequest{
		Update:     &configurator.Config{},
		FullResync: true,
	})
	if err != nil {
		logrus.Errorf("Failed to reset VPP Agent: %s", err)
	}
	logrus.Infof("Finished resetting VPP Agent")
	return nil
}

// Close implements the close handler
func (f *Flush) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if f.GetNext() != nil {
		return f.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
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
	_ = self.reset()

	return self
}
