// Copyright (c) 2020 Doc.ai, Inc and/or its affiliates.
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

package caddyfile

import (
	"flag"
	"os"
)

// Path parses corefile path from argument1s or returns default value
func Path() string {
	cl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	p := cl.String("conf", "/etc/coredns/Corefile", "")
	_ = cl.Parse(os.Args[1:])
	return *p
}
