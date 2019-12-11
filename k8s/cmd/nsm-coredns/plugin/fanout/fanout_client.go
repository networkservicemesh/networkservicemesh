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

package fanout

import (
	"crypto/tls"
	"runtime"
	"time"

	"github.com/coredns/coredns/plugin/pkg/up"
	"github.com/pkg/errors"
)

type fanoutClient struct {
	addr      string
	transport *Transport
	health    HealthChecker
}

func createFanoutClient(addr string) *fanoutClient {
	P := up.New()
	P.Start(time.Millisecond)

	a := &fanoutClient{
		addr:      addr,
		transport: newTransport(addr),
		health:    NewHealthChecker(addr),
	}
	runtime.SetFinalizer(a, (*fanoutClient).finalizer)
	return a
}

func (p *fanoutClient) setTLSConfig(cfg *tls.Config) {
	p.transport.setTLSConfig(cfg)
	p.health.SetTLSConfig(cfg)
}

func (p *fanoutClient) setExpire(expire time.Duration) {
	p.transport.setExpire(expire)
}

func (p *fanoutClient) healthCheck() error {
	if p.health == nil {
		return errors.New("no healthchecker")
	}
	return p.health.Check()
}

func (p *fanoutClient) finalizer() {
	p.transport.Stop()
}

func (p *fanoutClient) start() {
	p.transport.Start()
}
