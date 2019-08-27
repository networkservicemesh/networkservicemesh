package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/networkservicemesh/networkservicemesh/k8s/cmd/nsm-coredns/env"

	"github.com/caddyserver/caddy"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/k8s/cmd/nsm-coredns/api/update"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

func startUpdateServer() error {
	clientSockPath := env.UpdateAPIClientSock.StringValue()
	if clientSockPath == "" {
		return errors.New("client socket path can't be empty")
	}
	tools.SocketCleanup(clientSockPath)
	l, err := net.Listen("unix", clientSockPath)
	if err != nil {
		return err
	}
	server := tools.NewServer()
	update.RegisterDNSConfigServiceServer(server, newUpdateServer())
	go func() {
		fmt.Println("Update server started")
		err := server.Serve(l)
		fmt.Printf("An errur during serve update server: %v", err)
	}()
	return nil
}

func parseCorefilePath() string {
	cl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	path := cl.String("conf", caddy.DefaultConfigFile, "")
	cl.Parse(os.Args[1:])
	return *path
}

func newUpdateServer() update.DNSConfigServiceServer {
	result := &updateServer{
		dnsConfigManager: utils.NewDNSConfigManager(defaultBasicDNSConfig()),
		corefilePath:     parseCorefilePath(),
	}
	result.updateCorefileOperation = utils.NewSingleAsyncOperation(func() {
		file := result.dnsConfigManager.Caddyfile(result.corefilePath)
		err := file.Save()
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("Corefile updated")
		for _, instance := range caddy.Instances() {
			input, err := caddy.LoadCaddyfile(instance.Caddyfile().ServerType())
			if err != nil {
				fmt.Printf("An error %v during loading caddyfile\n,", err)
				continue
			}
			instance.Restart(input)
		}
		fmt.Println("Caddy servers restarted")
	})
	return result
}

func defaultBasicDNSConfig() connectioncontext.DNSConfig {
	return connectioncontext.DNSConfig{
		DnsServerIps:  []string{env.DefaultDNSServerIP.GetStringOrDefault("10.96.0.10")},
		SearchDomains: []string{},
	}
}

type updateServer struct {
	dnsConfigManager        *utils.DNSConfigManager
	corefilePath            string
	updateCorefileOperation utils.Operation
}

func (s *updateServer) AddDNSContext(ctx context.Context, msg *update.AddDNSContextMessage) (*empty.Empty, error) {
	for _, c := range msg.Context.Configs {
		fmt.Printf("Added new config %v with connection id %v", c, msg.ConnectionID)
		s.dnsConfigManager.Store(msg.ConnectionID, *c)
	}
	s.updateCorefileOperation.Run()
	return new(empty.Empty), nil
}

func (s *updateServer) RemoveDNSContext(ctx context.Context, msg *update.RemoveDNSContextMessage) (*empty.Empty, error) {
	s.dnsConfigManager.Delete(msg.ConnectionID)
	s.updateCorefileOperation.Run()
	return new(empty.Empty), nil
}
