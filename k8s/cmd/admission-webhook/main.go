package main

import (
	"crypto/tls"
	"fmt"

	"net/http"
	"os"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
)

func main() {
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	logrus.Info("Admission Webhook starting...")

	repo = os.Getenv(repoEnv)
	if repo == "" {
		repo = repoDefault
	}

	initContainer = os.Getenv(initContainerEnv)
	if initContainer == "" {
		initContainer = initContainerDefault
	}

	tag = os.Getenv(tagEnv)
	if tag == "" {
		tag = tagDefault
	}

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		logrus.Fatalf("Failed to load key pair: %v", err)
	}

	whsvr := &webhookServer{
		server: &http.Server{
			Addr:      fmt.Sprintf(":%v", 443),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.serve)
	whsvr.server.Handler = mux

	// start webhook server in new routine
	go func() {
		if err := whsvr.server.ListenAndServeTLS("", ""); err != nil {
			logrus.Fatalf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	logrus.Info("Server started")
	<-c
}
