package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

var version string

func main() {
	logrus.Info("Starting crossconnect-monitor...")
	log.Printf("Version: %v\n", version)
	utils.PrintAllEnv(logrus.StandardLogger())
	var wg sync.WaitGroup
	wg.Add(1)
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	go func() {
		<-c
		closing = true
		wg.Done()
	}()

	prom, err := tools.ReadEnvBool(metrics.PrometheusEnv, metrics.PrometheusDefault)
	if err == nil && prom {
		logrus.Infof("Starting Prometheus server")
		promServer := metrics.GetPrometheusMetricsServer()
		go func() {
			err = promServer.ListenAndServe()
			if err != nil {
				logrus.Errorf("failed to listen and serve prometheus server: %v", err)
			}
		}()

		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := promServer.Shutdown(ctx)
			if err != nil {
				logrus.Errorf("failed to shut down server: %v", err)
			}
		}()
	} else {
		logrus.Errorf("failed to read PROMETHEUS env var")
	}

	lookForNSMServers()

	wg.Wait()
}
