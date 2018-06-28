// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	injector "github.com/ligato/networkservicemesh/pkg/sidecarinjector"
)

func main() {

	port := flag.Int("port", 443, "Webhook server port")
	cert := flag.String("tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	key := flag.String("tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	sidecarConfig := flag.String("sidecarCfgFile", "/etc/webhook/config/sidecarconfig.yaml", "File containing the mutation configuration.")
	flag.Parse()

	pair, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		glog.Fatalf("Failed to load key pair: %v", err)
	}

	server := injector.Server{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", *port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
		SideCarConfigFile: *sidecarConfig,
	}

	if server.SideCarConfig, err = loadConfig(server.SideCarConfigFile); err != nil {
		glog.Fatalf("Failed to load config file %v err %v", server.SideCarConfigFile, err)
	}

	go func() {
		if err := server.Start(); err != nil {
			glog.Fatalf("Failed to start injector server err %v\n", err)
		}
	}()

	// blocking call
	handleExitSignal()

	glog.Info("Exit signal received clean up")
	server.Server.Shutdown(context.Background())
}

func handleExitSignal() {
	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
}

func loadConfig(configFile string) (*injector.Config, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	cfg := &injector.Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
