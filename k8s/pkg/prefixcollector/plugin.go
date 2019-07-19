package prefixcollector

import (
	"context"
	"net"
	"path"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/client-go/rest"
)

// StartPrefixPlugin creates an instance of a prefix plugin and registers it
func StartPrefixPlugin(config *rest.Config) error {
	endpoint := path.Join(plugins.PluginRegistryPath, "k8s-prefixes.sock")
	if err := tools.SocketCleanup(endpoint); err != nil {
		return err
	}

	if err := createPlugin(config, endpoint); err != nil {
		return err
	}

	conn, err := tools.SocketOperationCheck(tools.SocketPath(endpoint))
	if err != nil {
		return err
	}

	if err = conn.Close(); err != nil {
		return err
	}

	if err = registerPlugin(endpoint); err != nil {
		return err
	}

	return nil
}

func createPlugin(config *rest.Config, endpoint string) error {
	sock, err := net.Listen("unix", endpoint)
	if err != nil {
		return err
	}

	tracer := opentracing.GlobalTracer()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	service, err := newPrefixService(config)
	if err != nil {
		return err
	}

	plugins.RegisterConnectionPluginServer(server, service)

	go func() {
		if err := server.Serve(sock); err != nil {
			logrus.Error("Failed to start Prefix Plugin grpc server", endpoint, err)
		}
	}()

	return nil
}

func registerPlugin(endpoint string) error {
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial("unix:"+plugins.PluginRegistrySocket, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))
	defer func() { _ = conn.Close() }()
	if err != nil {
		logrus.Fatalf("Cannot connect to the service: %v", err)
	}

	client := plugins.NewPluginRegistryClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	_, err = client.Register(ctx, &plugins.PluginInfo{
		Endpoint: endpoint,
		Features: []plugins.PluginType{plugins.PluginType_CONNECTION},
	})
	if err != nil {
		return err
	}

	return nil
}
