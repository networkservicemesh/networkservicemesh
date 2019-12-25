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

// +build unit_test

package fanout

import (
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

func TestDnsClientClose(t *testing.T) {
	s := dnstest.NewServer(func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		checkErr(w.WriteMsg(ret))
	})
	defer s.Close()

	msg := new(dns.Msg)
	msg.SetQuestion("example.org.", dns.TypeA)
	req := request.Request{W: &test.ResponseWriter{}, Req: msg}

	for i := 0; i < 100; i++ {
		p := createFanoutClient(s.Addr)
		p.start()
		go func() { p.Connect(req) }()
		go func() { p.Connect(req) }()
	}
}

func TestProtocol(t *testing.T) {
	p := createFanoutClient("bad_address")

	req := request.Request{W: &test.ResponseWriter{TCP: true}, Req: new(dns.Msg)}

	go func() {
		p.Connect(req)
	}()

	proto := <-p.transport.dial
	p.transport.ret <- nil
	if proto != "tcp" {
		t.Errorf("Unexpected protocol in expected tcp, actual %q", proto)
	}
}
