package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/probes"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/probes/health"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var version string

func main() {
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	logrus.Info("Admission Webhook starting...")
	logrus.Infof("Version: %v", version)
	goals := &admissionWebhookGoals{}
	prob := probes.New("NSM Admission webhook probes", goals)
	prob.BeginHealthCheck()
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		logrus.Fatalf("Failed to load key pair: %v", err)
	}
	goals.SetKeyPairLoaded()
	addr := fmt.Sprintf(":%v", defaultPort)
	whsvr := &nsmAdmissionWebhook{
		server: &http.Server{
			Addr:      addr,
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.serve)
	whsvr.server.Handler = mux
	prob.Append(health.NewHttpServeMuxHealth(tools.NewAddr("https", addr), mux, time.Minute))
	// start webhook server in new routine
	go func() {
		if err := whsvr.server.ListenAndServeTLS("", ""); err != nil {
			logrus.Fatalf("Failed to listen and serve webhook server: %v", err)
		}
	}()
	goals.SetServerStarted()
	logrus.Info("Server started")
	<-c
}

func getRepo() string {
	repo := os.Getenv(repoEnv)
	if repo == "" {
		repo = repoDefault
	}
	return repo
}

func getTag() string {
	tag := os.Getenv(tagEnv)
	if tag == "" {
		tag = tagDefault
	}
	return tag
}

func getInitContainer() string {
	initContainer := os.Getenv(initContainerEnv)
	if initContainer == "" {
		initContainer = initContainerDefault
	}
	return initContainer
}

func getJaegerHost() string {
	return os.Getenv(jaegerHostEnv)
}

func getJaegerPort() string {
	return os.Getenv(jaegerPortEnv)
}
