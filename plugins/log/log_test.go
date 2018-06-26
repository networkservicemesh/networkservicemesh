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

package log_test

import (
	"github.com/ligato/networkservicemesh/plugins/log"
	. "github.com/onsi/gomega"
	"testing"
)

type Plugin struct {
	log.Logger
}

func TestNewLog(t *testing.T) {
	RegisterTestingT(t)
	p1n := "p1"
	l := log.SharedLog(p1n)
	p1 := &Plugin{l}
	Expect(l).ToNot(BeNil())
	Expect(p1.Name()).To(Equal(p1n))

	p2n := "p2"
	l = log.SharedLog(p2n, p1)
	p2 := &Plugin{l}
	Expect(l).ToNot(BeNil())
	Expect(p2.Name()).To(Equal(p1n + p2n))

	l = log.SharedLog(p2n, p1)
	Expect(l).To(Equal(p2.Logger))

}
