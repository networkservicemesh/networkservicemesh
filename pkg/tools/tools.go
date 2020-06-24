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
	"net"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// location of network namespace for a process
	netnsfile = "/proc/self/ns/net"
	// MaxSymLink is maximum length of Symbolic Link
	MaxSymLink = 8192
)

type addrImpl struct {
	addr    string
	network string
}

func (a *addrImpl) String() string {
	return a.addr
}

func (a *addrImpl) Network() string {
	return a.network
}

//NewAddr returns new net.Addr with network and address
func NewAddr(network, addr string) net.Addr {
	return &addrImpl{
		network: network,
		addr:    addr,
	}
}

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
	return "", errors.New("namespace is not found")
}

// GetCurrentPodNameFromHostname returns pod name a container is running in
// Note: Pod name is read from `/etc/hostname`. The same approach is used in
// kubelet implementation. There is risk that host name may be overwritten
// and in that case this should be considered when referring to pod name
func GetCurrentPodNameFromHostname() (string, error) {
	podName, err := os.Hostname()
	return podName, err
}

// SocketCleanup check for the presence of a stale socket and if it finds it, removes it.
func SocketCleanup(listenEndpoint string) error {
	fi, err := os.Stat(listenEndpoint)
	if err == nil && (fi.Mode()&os.ModeSocket) != 0 {
		if err := os.Remove(listenEndpoint); err != nil {
			return errors.Wrapf(err, "cannot remove listen endpoint %s", listenEndpoint)
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failure stat of socket file %s", listenEndpoint)
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

type NSUrl struct {
	NsName string
	Intf   string
	Params url.Values
}

func parseNSUrl(urlString string) (*NSUrl, error) {
	result := &NSUrl{}
	// Remove possible leading spaces from network service name
	urlString = strings.Trim(urlString, " ")
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	path := strings.Split(url.Path, "/")
	if len(path) > 2 {
		return nil, errors.New("Invalid NSUrl format")
	}
	if len(path) == 2 {
		if len(path[1]) > 15 {
			return nil, errors.New("Interface part cannot exceed 15 characters")
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
		return false, errors.WithMessage(err, "unable to clarify secure or insecure mode")
	}
	return insecure, nil
}
