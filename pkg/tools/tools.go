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
	"net"
	"os"
	"time"

	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"

	"regexp"
	"syscall"

	"google.golang.org/grpc"
)

const (
	// location of network namespace for a process
	netnsfile = "/proc/self/ns/net"
	// MaxSymLink is maximum length of Symbolic Link
	MaxSymLink = 8192
)

// GetCurrentNS discoveres the namespace of a running process and returns in a string.
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

// SocketCleanup check for the presense of a stale socket and if it finds it, removes it.
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

// SocketOperationCheck checks for liveness of a gRPC server socket.
func SocketOperationCheck(listenEndpoint string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := dial(ctx, listenEndpoint)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func dial(ctx context.Context, unixSocketPath string) (*grpc.ClientConn, error) {
	c, err := grpc.DialContext(ctx, unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	return c, err
}
func WaitForPortAvailable(ctx context.Context, protoType string, registryAddress string, interval time.Duration) error {
	if interval < 0 {
		return errors.New("interval must be positive")
	}
	for ; true; <-time.After(interval) {
		select {
		case <-ctx.Done():
			return errors.New("timeout waiting for: " + protoType + ":" + registryAddress)
		default:
			conn, err := net.Dial(protoType, registryAddress)
			if err != nil {
				logrus.Infof("Waiting for liveness probe: %s:%s", protoType, registryAddress)
				time.Sleep(interval)
				continue
			}
			conn.Close()
			return nil
		}
	}
	return nil
}
