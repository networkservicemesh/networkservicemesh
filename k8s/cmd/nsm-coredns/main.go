package main

import (
	"fmt"
	"os"
	"strings"

	env2 "github.com/networkservicemesh/networkservicemesh/k8s/cmd/nsm-coredns/env"

	"github.com/networkservicemesh/networkservicemesh/utils/caddyfile"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
	_ "github.com/coredns/coredns/plugin/bind"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/log"

	_ "github.com/networkservicemesh/networkservicemesh/k8s/cmd/nsm-coredns/plugin/fanout"
)

var version string

func init() {
	dnsserver.Directives = append(dnsserver.Directives, "fanout")
}

func main() {
	fmt.Println("Starting nsm-coredns")
	fmt.Printf("Version: %v\n", version)

	if env2.UseUpdateAPIEnv.GetBooleanOrDefault(false) {
		path := parseCorefilePath()
		file := caddyfile.NewCaddyfile(path)
		file.WriteScope(".").Write("log").Write(fmt.Sprintf("fanout %v", strings.Join(defaultBasicDNSConfig().DnsServerIps, " ")))
		err := file.Save()
		fmt.Println(file.String())
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(2)
		}
		fmt.Println("Starting dns context update server...")
		err = startUpdateServer()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(2)
		}
	}

	coremain.Run()
}
