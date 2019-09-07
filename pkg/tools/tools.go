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

package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-errors/errors"
	"github.com/opentracing/opentracing-go"
	pkgerrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

const (
	// location of network namespace for a process
	netnsfile = "/proc/self/ns/net"
	// MaxSymLink is maximum length of Symbolic Link
	MaxSymLink = 8192
)

// GetCurrentNS discovers the namespace of a running process and returns in a string.
func GetCurrentNS() (string, error) {
	buf := make([]byte, MaxSymLink)
	numBytes, err := syscall.Readlink(netnsfile, buf)
	if err != nil {
		return "", err
	}
	link := string(buf[0:numBytes])
	nsRegExp := regexp.MustCompile("net:\\[(.*)\\]")
	submatches := nsRegExp.FindStringSubmatch(link)
	if len(submatches) >= 1 {
		return submatches[1], nil
	}
	return "", fmt.Errorf("namespace is not found")
}

// SocketCleanup check for the presence of a stale socket and if it finds it, removes it.
func SocketCleanup(listenEndpoint string) error {
	fi, err := os.Stat(listenEndpoint)
	if err == nil && (fi.Mode()&os.ModeSocket) != 0 {
		if err := os.Remove(listenEndpoint); err != nil {
			return fmt.Errorf("cannot remove listen endpoint %s with error: %+v", listenEndpoint, err)
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failure stat of socket file %s with error: %+v", listenEndpoint, err)
	}
	return nil
}

// Unix socket file path.
type SocketPath string

func (socket SocketPath) Network() string {
	return "unix"
}

func (socket SocketPath) String() string {
	return string(socket)
}

func NewOSSignalChannel() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		os.Interrupt,
		// More Linux signals here
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	return c
}

// WaitForPortAvailable waits while the port will is available. Throws exception if the context is done.
func WaitForPortAvailable(ctx context.Context, protoType, registryAddress string, idleSleep time.Duration) error {
	if idleSleep < 0 {
		return errors.New("idleSleep must be positive")
	}
	logrus.Infof("Waiting for liveness probe: %s:%s", protoType, registryAddress)
	last := time.Now()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout waiting for: " + protoType + ":" + registryAddress)
		default:
			var d net.Dialer
			conn, err := d.DialContext(ctx, protoType, registryAddress)
			if conn != nil {
				_ = conn.Close()
			}
			if err == nil {
				return nil
			}
			if time.Since(last) > time.Minute {
				logrus.Infof("Waiting for liveness probe: %s:%s", protoType, registryAddress)
				last = time.Now()
			}
			// Sleep to not overflow network
			<-time.After(idleSleep)
		}
	}
}

func parseKV(kv, kvsep string) (string, string) {
	keyValue := strings.Split(kv, kvsep)
	if len(keyValue) != 2 {
		keyValue = []string{"", ""}
	}
	return strings.Trim(keyValue[0], " "), strings.Trim(keyValue[1], " ")
}

// ParseKVStringToMap parses the input string
func ParseKVStringToMap(input, sep, kvsep string) map[string]string {
	result := map[string]string{}
	pairs := strings.Split(input, sep)
	for _, pair := range pairs {
		k, v := parseKV(pair, kvsep)
		result[k] = v
	}
	return result
}

// initJaeger returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func InitJaeger(service string) (opentracing.Tracer, io.Closer) {
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

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

type NSUrl struct {
	NsName string
	Intf   string
	Params url.Values
}

func parseNSUrl(urlString string) (*NSUrl, error) {
	result := &NSUrl{}

	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	path := strings.Split(url.Path, "/")
	if len(path) > 2 {
		return nil, fmt.Errorf("Invalid NSUrl format")
	}
	if len(path) == 2 {
		if len(path[1]) > 15 {
			return nil, fmt.Errorf("Interface part cannot exceed 15 characters")
		}
		result.Intf = path[1]
	}
	result.NsName = path[0]
	result.Params = url.Query()
	return result, nil
}

func ParseAnnotationValue(value string) ([]*NSUrl, error) {
	var result []*NSUrl
	urls := strings.Split(value, ",")
	for _, u := range urls {
		nsurl, err := parseNSUrl(u)
		if err != nil {
			return nil, err
		}
		result = append(result, nsurl)
	}
	return result, nil
}

// ReadEnvBool reads environment variable and treat it as bool
func ReadEnvBool(env string, value bool) (bool, error) {
	str := os.Getenv(env)
	if str == "" {
		return value, nil
	}

	return strconv.ParseBool(str)
}

// IsInsecure checks environment variable INSECURE
func IsInsecure() (bool, error) {
	insecure, err := ReadEnvBool(InsecureEnv, insecureDefault)
	if err != nil {
		return false, pkgerrors.WithMessage(err, "unable to clarify secure or insecure mode")
	}
	return insecure, nil
}
