package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

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
	err := waitForCorefile()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Version: %v\n", version)
	coremain.Run()
}

func waitForCorefile() error {
	const timeoutDuration = time.Second * 5
	timeout := time.After(timeoutDuration)
	path := pathToCorefile()
	for {
		_, err := os.Stat(path)
		if os.IsExist(err) {
			return nil
		}
		select {
		case <-timeout:
			return errors.New(fmt.Sprintf("corefile not found. Path: %v", path))
		case <-time.After(time.Millisecond * 500):
			break
		}
	}
}

func pathToCorefile() string {
	cl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	path := cl.String("conf", caddy.DefaultConfigFile, "")
	cl.Parse(os.Args[1:])
	return *path
}
