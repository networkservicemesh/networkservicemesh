// Copyright (c) 2019 Cisco Systems, Inc and/or its affiliates.
//
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

// +build  performance

package fanout

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/transport"

	"github.com/coredns/coredns/plugin/forward"

	"github.com/coredns/coredns/plugin"

	"github.com/coredns/coredns/plugin/test"

	"github.com/pkg/errors"

	"github.com/miekg/dns"
)

type testServerType rune

const (
	WorkingServer    testServerType = 'g'
	UnreachableSerer testServerType = 'e'
)

func samples() [][]testServerType {
	return [][]testServerType{
		{WorkingServer},
		{WorkingServer, UnreachableSerer},
		{UnreachableSerer, WorkingServer},
		{UnreachableSerer, UnreachableSerer, WorkingServer},
	}
}
func BenchmarkFanoutPlugin(b *testing.B) {
	samples := samples()
	for i := 0; i < b.N; i++ {
		for _, sample := range samples {
			f := NewFanout()
			benchmarkSample(b, f, func(addr string) {
				f.addClient(createFanoutClient(addr))
			}, sample)
		}
	}
}
func BenchmarkForwardPlugin(b *testing.B) {
	samples := samples()
	for i := 0; i < b.N; i++ {
		for _, sample := range samples {
			f := forward.New()
			benchmarkSample(b, f, func(addr string) {
				f.SetProxy(forward.NewProxy(addr, transport.DNS))
			}, sample)
		}
	}
}

func benchmarkSample(b *testing.B, handler plugin.Handler, add func(string), sample []testServerType) {
	b.StopTimer()
	writer := &cachedDNSWriter{ResponseWriter: new(test.ResponseWriter)}
	writer.TCP = true
	closeFunc := preparePlugin(add, sample)
	req := new(dns.Msg)
	req.SetQuestion("test.", dns.TypeA)
	b.StartTimer()
	_, err := handler.ServeDNS(context.Background(), writer, req)
	if err != nil {
		b.Fatal(err.Error())
	}
	if writer.answers[0].Rcode != dns.RcodeSuccess {
		b.Fatal("answer should be a success")
	}
	b.StopTimer()
	closeFunc()
	b.StartTimer()
}

func preparePlugin(addClient func(string), sample []testServerType) (clearFunc func()) {
	var servers []*server
	for _, r := range sample {
		s := createServerByType(testServerType(r))
		servers = append(servers, s)
		addClient(s.Addr)
	}
	return func() {
		for _, s := range servers {
			s.close()
		}
	}
}

func createServerByType(t testServerType) *server {
	switch t {
	case WorkingServer:
		return testServer()
	case UnreachableSerer:
		return dummyServer()
	}
	panic(errors.New(fmt.Sprintf("unknown server type %v", t)))
}

func testServer() *server {
	return newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "test." {
			msg := dns.Msg{
				Answer: []dns.RR{makeRecordA("example1. 3600	IN	A 10.0.0.1")},
			}
			msg.SetReply(r)
			w.WriteMsg(&msg)
		}
	})
}

func dummyServer() *server {
	return newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		<-time.After(time.Millisecond * 100)
	})
}
