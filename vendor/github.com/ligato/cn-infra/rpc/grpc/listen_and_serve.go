// Copyright (c) 2017 Cisco and/or its affiliates.
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

package grpc

import (
	"fmt"
	"io"
	"net"
	"time"

	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc"
)

// ListenAndServe is a function that uses <config> & <handler> to handle
// GRPC Requests.
// It return an instance of io.Closer to close the Server (net listener) during cleanup.
type ListenAndServe func(config Config, grpcServer *grpc.Server) (
	netListener io.Closer, err error)

// FromExistingServer is used mainly for testing purposes
func FromExistingServer(listenAndServe ListenAndServe) *Plugin {
	return &Plugin{listenAndServe: listenAndServe}
}

// ListenAndServeGRPC starts a netListener.
func ListenAndServeGRPC(config *Config, grpcServer *grpc.Server) (netListener net.Listener, err error) {

	// Default to tcp socket type of not specified for backward compatibility
	socketType := config.Network
	if socketType == "" {
		socketType = "tcp"
	}

	if socketType == "unix" || socketType == "unixpacket" {
		permissions, err := getUnixSocketFilePermissions(config.Permission)
		if err != nil {
			return nil, err
		}
		if err := checkUnixSocketFileAndDirectory(config.Endpoint, config.ForceSocketRemoval); err != nil {
			return nil, err
		}

		netListener, err = net.Listen(socketType, config.Endpoint)
		if err != nil {
			return nil, err
		}

		// Set permissions to the socket file
		if err := os.Chmod(config.Endpoint, permissions); err != nil {
			return nil, err
		}
	} else {
		netListener, err = net.Listen(socketType, config.Endpoint)
		if err != nil {
			return nil, err
		}
	}

	var errCh chan error
	go func() {
		if err := grpcServer.Serve(netListener); err != nil {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	select {
	case err := <-errCh:
		return nil, err
		// Wait 100ms to create a new stream, so it doesn't bring too much
		// overhead when retry.
	case <-time.After(100 * time.Millisecond):
		//everything is probably fine
		return netListener, nil
	}
}

// Resolve permissions and return FileMode
func getUnixSocketFilePermissions(permissions int) (os.FileMode, error) {
	if permissions > 0 {
		if permissions > 7777 {
			return 0, fmt.Errorf("incorrect unix socket file/path permission value '%d'", permissions)
		}
		// Convert to correct mode format
		mode, err := strconv.ParseInt(strconv.Itoa(permissions), 8, 32)
		if err != nil {
			return 0, fmt.Errorf("failed to parse socket file permissions %d", permissions)
		}
		return os.FileMode(mode), nil
	}
	return os.ModePerm, nil
}

// Check old socket file/directory of the unix domain socket. Remove old socket file if exists or create the directory
// path if does not exist.
func checkUnixSocketFileAndDirectory(endpoint string, forceRemoval bool) error {
	_, err := os.Stat(endpoint)
	if err == nil && forceRemoval {
		// Remove old socket file if required
		if err := os.Remove(endpoint); err != nil {
			return err
		}
	}
	if os.IsNotExist(err) {
		// Create the directory
		lastIdx := strings.LastIndex(endpoint, "/")
		path := endpoint[:lastIdx]
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}
