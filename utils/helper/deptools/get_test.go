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

package deptools_test

import (
	"testing"

	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	. "github.com/onsi/gomega"
)

func TestGet(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{
		Deps: NonOptionalDeps{
			One: &Plugin{},
		},
	}
	d, err := deptools.Get(p)
	Expect(err).To(Succeed())
	Expect(d == p.Deps).To(BeTrue())
}

func TestGetNoDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &Plugin{}
	d, err := deptools.Get(p)
	Expect(err).To(Succeed())
	Expect(d).To(BeNil())
}
