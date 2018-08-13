// Copyright (c) 2018 Cisco and/or its affiliates.
// Copyright 2018 vArmour Networks.
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
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ghodss/yaml"
	injector "github.com/ligato/networkservicemesh/pkg/sidecarinjector"
	"github.com/ligato/networkservicemesh/plugins/logger"
)

const (
	loggerName        = "sidecarInjector"
	serverPort        = 443
	webhookCertPath   = "/etc/webhook/certs/cert.pem"
	webhookKeyPath    = "/etc/webhook/certs/key.pem"
	sidecarConfigPath = "/etc/webhook/config/sidecarconfig.yaml"
)

func main() {

	port := flag.Int("port", serverPort, "Webhook server port")
	cert := flag.String("tlsCertFile", webhookCertPath, "File containing the x509 Certificate for HTTPS.")
	key := flag.String("tlsKeyFile", webhookKeyPath, "File containing the x509 private key to --tlsCertFile.")
	sidecarConfig := flag.String("sidecarCfgFile", sidecarConfigPath, "File containing the mutation configuration.")
	flag.Parse()

	pair, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		panic(fmt.Sprintf("Failed to load key pair: %v", err))
	}

	server := injector.Server{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", *port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
		SideCarConfigFile: *sidecarConfig,
		Log:               logger.ByName(loggerName),
	}

	if server.SideCarConfig, err = loadConfig(server.SideCarConfigFile); err != nil {
		server.Log.Fatalf("Failed to load config file %v err %v", server.SideCarConfigFile, err)
	}

	go func() {
		if err := server.Start(); err != nil {
			server.Log.Fatalf("Failed to start injector server err %v\n", err)
		}
	}()

	// blocking call
	handleExitSignal()

	server.Log.Info("Exit signal received clean up")
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
	cfg := &injector.Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
