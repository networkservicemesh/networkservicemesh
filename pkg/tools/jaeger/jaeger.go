package jaeger

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

const (
	opentracingEnv     = "TRACER_ENABLED"
	opentracingDefault = false
)

// IsOpentracingEnabled returns true if opentracing enabled
func IsOpentracingEnabled() bool {
	val, err := readEnvBool(opentracingEnv, opentracingDefault)
	if err == nil {
		return val
	}
	return opentracingDefault
}

func readEnvBool(env string, value bool) (bool, error) {
	str := os.Getenv(env)
	if str == "" {
		return value, nil
	}

	return strconv.ParseBool(str)
}

type emptyCloser struct {
}

func (*emptyCloser) Close() error {
	// Ignore
	return nil
}

// InitJaeger -  returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func InitJaeger(service string) io.Closer {
	if !IsOpentracingEnabled() {
		return &emptyCloser{}
	}
	if opentracing.IsGlobalTracerRegistered() {
		logrus.Warningf("global opentracer is already initialized")
	}
	cfg, err := config.FromEnv()
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot create Jaeger configuration: %v\n", err))
	}

	if cfg.ServiceName == "" {
		var hostname string
		hostname, err = os.Hostname()
		if err == nil {
			cfg.ServiceName = fmt.Sprintf("%s@%s", service, hostname)
		} else {
			cfg.ServiceName = service
		}
	}
	if cfg.Sampler.Type == "" {
		cfg.Sampler.Type = "const"
	}
	if cfg.Sampler.Param == 0 {
		cfg.Sampler.Param = 1
	}
	if !cfg.Reporter.LogSpans {
		cfg.Reporter.LogSpans = true
	}

	logrus.Infof("Creating logger from config: %v", cfg)
	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		logrus.Errorf("ERROR: cannot init Jaeger: %v\n", err)
		return &emptyCloser{}
	}
	opentracing.SetGlobalTracer(tracer)
	return closer
}
