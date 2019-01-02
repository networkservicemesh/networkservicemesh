package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

// initJaeger returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func initJaeger(service string) (opentracing.Tracer, io.Closer) {
	jaegerHost := os.Getenv("JAEGER_SERVICE_HOST")
	jaegerPort := os.Getenv("JAEGER_SERVICE_PORT_JAEGER")
	jaegerHostPort := fmt.Sprintf("%s:%s", jaegerHost, jaegerPort)

	logrus.Infof("Using Jaeger host/port: %s", jaegerHostPort)

	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: jaegerHostPort,
		},
	}
	tracer, closer, err := cfg.New(service, config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func main() {

	tracer, closer := initJaeger("nsmd")

	apiRegistry := nsmd.NewApiRegistry()
	serviceRegistry := nsmd.NewServiceRegistry()

	model := model.NewModel() // This is TCP gRPC server uri to access this NSMD via network.

	defer serviceRegistry.Stop()

	if err := nsmd.StartDataplaneRegistrarServer(model); err != nil {
		logrus.Fatalf("Error starting dataplane service: %+v", err)
		os.Exit(1)
	}

	if err := nsmd.StartNSMServer(model, serviceRegistry, apiRegistry, tracer); err != nil {
		logrus.Fatalf("Error starting nsmd service: %+v", err)
		os.Exit(1)
	}

	if err := nsmd.StartAPIServer(model, apiRegistry, serviceRegistry); err != nil {
		logrus.Fatalf("Error starting nsmd api service: %+v", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		wg.Done()
	}()
	wg.Wait()

	closer.Close()
}
