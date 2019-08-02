package prefixcollector

import (
	"context"
	"net"
	"path"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/client-go/rest"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
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

	if err := registerPlugin(endpoint); err != nil {
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
	conn, err := tools.DialUnix(plugins.PluginRegistrySocket)
	if err != nil {
		logrus.Fatalf("Cannot connect to the Plugin Registry: %v", err)
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			logrus.Fatalf("Cannot close the connection to the Plugin Registry: %v", err)
		}
	}()

	client := plugins.NewPluginRegistryClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	_, err = client.Register(ctx, &plugins.PluginInfo{
		Endpoint:     endpoint,
		Capabilities: []plugins.PluginCapability{plugins.PluginCapability_CONNECTION},
	})
	if err != nil {
		return err
	}

	return nil
}
