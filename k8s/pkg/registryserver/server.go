package registryserver

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

func New(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Panicln(err)
	}

	defer listener.Close()
	server := grpc.NewServer()
	registry.RegisterNetworkServiceRegistryServer(server, &registryService{})
	return server.Serve(listener)
}
