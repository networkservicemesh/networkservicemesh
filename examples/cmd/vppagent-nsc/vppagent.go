package main

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/converter"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func CreateVppInterface(nscConnection *connection.Connection, baseDir string, vppAgentEndpoint string) error {
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(vppAgentEndpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
	client := rpc.NewDataChangeServiceClient(conn)

	conversionParameters := &converter.ConnectionConversionParameters{
		Name:      "SRC-" + nscConnection.GetId(),
		Terminate: true,
		Side:      converter.SOURCE,
		BaseDir:   baseDir,
	}
	dataChange, err := converter.NewMemifInterfaceConverter(nscConnection, conversionParameters).ToDataRequest(nil, true)

	if err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Infof("Sending DataChange to vppagent: %v", dataChange)
	if _, err := client.Put(context.Background(), dataChange); err != nil {
		logrus.Error(err)
		client.Del(context.Background(), dataChange)
		return err
	}
	return nil
}

func Reset(vppAgentEndpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	tools.WaitForPortAvailable(ctx, "tcp", vppAgentEndpoint, 100*time.Millisecond)

	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(vppAgentEndpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()
	client := rpc.NewDataResyncServiceClient(conn)
	logrus.Infof("Resetting vppagent...")
	_, err = client.Resync(context.Background(), &rpc.DataRequest{})
	if err != nil {
		logrus.Errorf("failed to reset vppagent: %s", err)
	}
	logrus.Infof("Finished resetting vppagent...")
	return nil
}
