package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/networkservicemesh/networkservicemesh/utils/caddyfile"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
	_ "github.com/coredns/coredns/plugin/bind"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/reload"

	_ "github.com/networkservicemesh/networkservicemesh/k8s/cmd/nsm-coredns/plugin/fanout"
)

var version string

func init() {
	dnsserver.Directives = append(dnsserver.Directives, "fanout")
}

func main() {
	fmt.Println("Starting nsm-coredns...")
	fmt.Printf("Version: %v\n", version)
	path := pathToCorefile()
	caddyfile := caddyfile.NewCaddyfile(path)
	caddyfile.WriteScope(".").Write("reload")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("used default corefile")
		caddyfile.Save()
	}
	coremain.Run()
}

func pathToCorefile() string {
	cl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	path := cl.String("conf", caddy.DefaultConfigFile, "")
	cl.Parse(os.Args[1:])
	return *path
}
