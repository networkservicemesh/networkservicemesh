package plugins

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type ConnectionPluginRegistry interface {
	pluginManager
	UpdateConnection(connection.Connection)
	ValidateConnection(connection.Connection) error
}

type connectionPluginRegistry struct {
	connectionPlugins []plugins.ConnectionPluginClient
}

func (cpr *connectionPluginRegistry) registerPlugin(endpoint string) error {
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))
	//defer conn.Close() // TODO: store a connection and close it when registry.Stop() is called
	if err != nil {
		return err
	}

	client := plugins.NewConnectionPluginClient(conn)
	cpr.connectionPlugins = append(cpr.connectionPlugins, client)
	return nil
}

func (cpr *connectionPluginRegistry) UpdateConnection(conn connection.Connection) {
	connCtx := conn.GetContext()
	for _, plugin := range cpr.connectionPlugins {
		ctx, cancel := context.WithTimeout(context.Background(), 100 * time.Second)
		defer cancel() // TODO: fix defer in a loop

		var err error
		connCtx, err = plugin.UpdateConnectionContext(ctx, connCtx)
		if err != nil {
			logrus.Errorf("Connection Plugin returned an error: %v", err)
		}
	}
	conn.SetContext(connCtx)
}

func (cpr *connectionPluginRegistry) ValidateConnection(conn connection.Connection) error {
	for _, plugin := range cpr.connectionPlugins {
		ctx, cancel := context.WithTimeout(context.Background(), 100 * time.Second)
		defer cancel() // TODO: fix defer in a loop

		result, err := plugin.ValidateConnectionContext(ctx, conn.GetContext())
		if err != nil {
			logrus.Errorf("Connection Plugin returned an error: %v", err)
		}

		if result.GetStatus() != plugins.ConnectionValidationStatus_SUCCESS {
			return fmt.Errorf(result.GetErrorMessage())
		}
	}
	return nil
}
