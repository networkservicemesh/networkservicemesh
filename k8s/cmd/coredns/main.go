package main

import (
	"fmt"
	_ "github.com/coredns/coredns/plugin/bind"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/networkservicemesh/networkservicemesh/k8s/cmd/coredns/plugin/fanout"

	"github.com/coredns/coredns/coremain"
)

var version string

func main() {
	fmt.Printf("Version: %v\n", version)
	coremain.Run()
}
