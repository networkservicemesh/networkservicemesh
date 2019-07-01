package vppagent

import (
	"context"
	"fmt"
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
	"time"
)

type VppAgentFlush struct {
	endpoint.BaseCompositeEndpoint
	VppAgentEndpoint string
}

func (vf *VppAgentFlush) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if vf.GetNext() == nil {
		logrus.Fatal("The VPP Agent Flush composite requires that there is Next set")
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
	}
	if err := tools.WaitForPortAvailable(ctx, "tcp", vf.VppAgentEndpoint, 100*time.Millisecond); err != nil {
		return nil, err
	}

	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(vf.VppAgentEndpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	if err != nil {
		logrus.Errorf("Can't dial grpc server: %v", err)
		return nil, err
	}
	defer conn.Close()
	client := configurator.NewConfiguratorClient(conn)

	incomingConnection, err := vf.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	opaque := vf.GetNext().GetOpaque(incomingConnection)
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

func (vf *VppAgentFlush) Reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := tools.WaitForPortAvailable(ctx, "tcp", vf.VppAgentEndpoint, 100*time.Millisecond); err != nil {
		return err
	}
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(vf.VppAgentEndpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	if err != nil {
		logrus.Errorf("Can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
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
func (vf *VppAgentFlush) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if vf.GetNext() != nil {
		return vf.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// NewVppAgentFlush creates a VppAgentFlush
func NewVppAgentFlush(configuration *common.NSConfiguration, endpoint string) *VppAgentFlush {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	self := &VppAgentFlush{
		VppAgentEndpoint: endpoint,
	}
	self.Reset()

	return self
}
