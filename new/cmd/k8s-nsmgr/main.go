package k8s_nsmgr

import (
	"net"
	"net/url"
	"path"

	"github.com/networkservicemesh/networkservicemesh/new/cmd/k8s-nsmgr/pkg/deviceplugin"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

var version string

// Default values and environment variables of proxy connection
const (
	ServerSock            = "networkservicemesh.io.sock"
	defaultConfigFilename = "/etc/k8s-nsmgr.conf"
)

func main() {
	// Capture signals to cleanup before exiting
	logrus.Info("Starting nsmd...")
	logrus.Infof("Version: %v", version)
	c := tools.NewOSSignalChannel()
	closer := jaeger.InitJaeger("k8s-nsmgr")
	defer func() { _ = closer.Close() }()

	// TODO - get config filename from viper
	viper.SetConfigFile(defaultConfigFilename)
	u, err := url.Parse(viper.GetString("registry_url"))
	if err != nil {
		logrus.Fatalf("registry_url improperly formated: %+v", err)
	}
	registryCC, err := tools.DialUrl(u)

	dp := deviceplugin.NewServer(viper.GetString("name"), viper.GetBool("insecure"), registryCC)
	listenEndpoint := path.Join(pluginapi.DevicePluginPath, ServerSock)
	sock, err := net.Listen("unix", listenEndpoint)
	if err != nil {
		logrus.Fatalf("failed to listen on %s: %+v", listenEndpoint, err)
	}
	grpcServer := tools.NewServerInsecure()
	pluginapi.RegisterDevicePluginServer(grpcServer, dp)
	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Error("failed to start device plugin grpc server", listenEndpoint, err)
		}
	}()

	<-c

}
