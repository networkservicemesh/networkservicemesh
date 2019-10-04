package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/caddyserver/caddy"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/k8s/api/nsm-coredns/update"
	"github.com/networkservicemesh/networkservicemesh/k8s/cmd/nsm-coredns/env"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

func startUpdateServer() error {
	clientSockPath := env.UpdateAPIClientSock.StringValue()
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

func updateResolvConfFile() {
	r := resolvConfFile{path: resolvConfFilePath}
	properties := []resolvConfProperty{
		{nameserverProperty, []string{"127.0.0.1"}},
		{searchProperty, r.Searches()},
		{optionsProperty, r.Options()},
	}
	r.ReplaceProperties(properties)
}

func defaultBasicDNSConfig() connectioncontext.DNSConfig {
	r := resolvConfFile{path: resolvConfFilePath}
	return connectioncontext.DNSConfig{
		DnsServerIps:  r.Nameservers(),
		SearchDomains: r.Searches(),
	}
}
