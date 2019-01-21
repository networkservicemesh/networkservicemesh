// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
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

package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

/*
	The example proxy-nsc is an implementation of HTTP proxy as NS Client.
	It can be use as an NS ingress proxy, where the proxy service is exposed as an
	outside facing service. Upon HTTP connection request the proxy will create
	a new NS connection. The connection is configured by env variables as an usual
	NS Client. The proxy will scan the HTTP request for headers of format:
		NSM-Lable:Value
	These would be transformed to NS Client request labels.

                       +------------+                      +-------------+
  GET / HTTP/1.1       |            |                      |             |
  NSM-App: Firewall    |            |     app=firewall     |             |
+----------------------> Proxy NSC  +----------------------> NS Endopint |
                       |            |                      |             |
                       |            |                      |             |
                       +------------+                      +-------------+
*/

const (
	proxyHostEnv      = "PROXY_HOST"
	defaultProxyHost  = ":8080"
	proxyHeaderPrefix = "nsm-"
)

var state struct {
	sync.RWMutex
	interfaceID int
	client      *client.NsmClient
}

func nsmDirector(req *http.Request) {
	state.Lock()
	defer state.Unlock()

	// Convert the
	state.client.OutgoingNscLabels = make(map[string]string)
	for name, headers := range req.Header {
		name = strings.ToLower(name)
		if strings.HasPrefix(name, proxyHeaderPrefix) {
			name = strings.TrimPrefix(name, proxyHeaderPrefix)
			state.client.OutgoingNscLabels[name] = strings.ToLower(headers[0])
		}
	}

	ifname := "nsm" + strconv.Itoa(state.interfaceID)
	state.interfaceID = state.interfaceID + 1

	outgoing, err := state.client.Connect(ifname, "kernel", "Primary interface")
	if err != nil {
		// cancel request
		logrus.Errorf("Error: %v", err)
		return
	}

	ipv4Addr, _, err := net.ParseCIDR(outgoing.GetContext().GetDstIpAddr())
	if err != nil {
		log.Fatal(err)
	}

	req.URL.Scheme = "http"
	req.URL.Host = ipv4Addr.String()
	req.URL.Path = "/"
	req.Host = req.URL.Host

	// TODO: NsmClient has a bug that it does return the connection before it is actually UP
	// Sleep for half a second, to give it time to finish the connection setup
	time.Sleep(500 * time.Millisecond)

	go func() {
		<-req.Context().Done()
		logrus.Infof("Connection goes down for: %v", outgoing)
		state.client.Close(outgoing)
	}()
}

func proxyHost() string {
	proxyHost, ok := os.LookupEnv(proxyHostEnv)
	if !ok {
		proxyHost = defaultProxyHost
	}
	return proxyHost
}

func main() {
	// Init the tracer
	tracer, closer := tools.InitJaeger("nsc")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	// Create the NSM client
	state.interfaceID = 0
	client, err := client.NewNSMClient(nil, nil)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client %v", err)
	}
	state.client = client

	// Create the reverse proxy
	reverseProxy := httputil.NewSingleHostReverseProxy(&url.URL{})
	reverseProxy.Director = nsmDirector

	logrus.Infof("Listen and Serve on %v", proxyHost())
	err = http.ListenAndServe(proxyHost(), reverseProxy)
	if err != nil {
		logrus.Errorf("Listen and serve failed with error: %v", err)
	}
}
