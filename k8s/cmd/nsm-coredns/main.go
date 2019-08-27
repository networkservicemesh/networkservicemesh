package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/reload"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils"

	"github.com/networkservicemesh/networkservicemesh/utils/caddyfile"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
	_ "github.com/coredns/coredns/plugin/bind"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/log"

	_ "github.com/networkservicemesh/networkservicemesh/k8s/cmd/nsm-coredns/plugin/fanout"
)

var version string
var pathToCoreFile string
var useReloadServer bool

func init() {
	dnsserver.Directives = append(dnsserver.Directives, "fanout")
	cl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	path := cl.String("conf", caddy.DefaultConfigFile, "")
	cl.Parse(os.Args[1:])
	pathToCoreFile = *path
}

type reloadServer struct {
	pathToCoreFile string
}

func (r *reloadServer) Reload(ctx context.Context, msg *reload.ReloadMessage) (*empty.Empty, error) {
	fmt.Println(fmt.Sprintf("recived reload msg %v", msg))
	ioutil.WriteFile(pathToCoreFile, []byte(msg.Context), os.ModePerm)
	instance := caddy.Instances()[0]
	input, err := caddy.LoadCaddyfile(instance.Caddyfile().ServerType())
	if err != nil {
		fmt.Println(err)
	}
	for _, inst := range caddy.Instances() {
		fmt.Println(inst.Caddyfile().ServerType())
	}
	caddy.Instances()[0].Restart(input)
	fmt.Println("caddy server restarted")
	return new(empty.Empty), nil
}

func main() {
	fmt.Println("Starting nsm-coredns...")
	fmt.Printf("Version: %v\n", version)
	path := pathToCoreFile
	useReloadServer, _ = strconv.ParseBool(utils.EnvVar("RELOAD").StringValue())
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file := caddyfile.NewCaddyfile(path)
		file.WriteScope(".:53")
		file.Save()
	}

	if useReloadServer {
		err := startReloadServer(path)
		fmt.Println(err)

	}

	coremain.Run()
}

func startReloadServer(pathToCorefile string) error {
	pathToSocket := path.Dir(pathToCorefile) + "/client.sock"
	sock, err := net.Listen("unix", pathToSocket)
	if err != nil {
		return err
	}
	server := tools.NewServer()
	reload.RegisterReloadServiceServer(server, &reloadServer{})
	go func() {
		fmt.Println("Reload server started")
		if err := server.Serve(sock); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}()
	return nil

}
