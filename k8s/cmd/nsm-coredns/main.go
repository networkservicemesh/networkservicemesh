package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
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
	fmt.Println("Starting nsm-coredns...")
	fmt.Printf("Version: %v\n", version)
	err := waitForCorefile()
	if err != nil {
		logrus.Error(err)
	}
	coremain.Run()
}

func waitForCorefile() error {
	const timeoutDuration = time.Second * 15
	path := pathToCorefile()
	timeout := time.After(timeoutDuration)
	for {
		_, err := os.Stat(path)
		if err == nil {
			return nil
		}
		select {
		case <-timeout:
			return errors.New(fmt.Sprintf("corefile not found. Path: %v", path))
		default:
			<-time.After(time.Millisecond * 1000)
		}
	}
}

func pathToCorefile() string {
	cl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	path := cl.String("conf", caddy.DefaultConfigFile, "")
	cl.Parse(os.Args[1:])
	return *path
}
