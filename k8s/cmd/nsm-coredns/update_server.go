package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/networkservicemesh/networkservicemesh/k8s/api/nsm-coredns/update"
	env2 "github.com/networkservicemesh/networkservicemesh/k8s/cmd/nsm-coredns/env"

	"github.com/caddyserver/caddy"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

func startUpdateServer() error {
	clientSockPath := env2.UpdateAPIClientSock.StringValue()
	if clientSockPath == "" {
		return errors.New("client socket path can't be empty")
	}
	err := tools.SocketCleanup(clientSockPath)
	if err != nil {
		return err
	}
	l, err := net.Listen("unix", clientSockPath)
	if err != nil {
		return err
	}
	server := tools.NewServer(context.Background())
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
	_ = cl.Parse(os.Args[1:])
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
			_, err = instance.Restart(input)
			if err != nil {
				fmt.Printf("An error %v during server instance restart\n", err)
			}
		}
		fmt.Println("Caddy servers restarted")
	})
	return result
}

func defaultBasicDNSConfig() connectioncontext.DNSConfig {
	return connectioncontext.DNSConfig{
		DnsServerIps:  env2.DefaultDNSServerIPList.GetStringListValueOrDefault("10.96.0.10"),
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
