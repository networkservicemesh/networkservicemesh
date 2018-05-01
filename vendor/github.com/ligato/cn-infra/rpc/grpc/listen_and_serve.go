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
	"io"
	"net"
	"time"

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
func ListenAndServeGRPC(config Config, grpcServer *grpc.Server) (netListener net.Listener, err error) {
	netListener, err = net.Listen("tcp", config.Endpoint)
	if err != nil {
		return nil, err
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
