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

package main

import (
	"fmt"
	"os"

	"github.com/ligato/cn-infra/core"
	"github.com/ligato/networkservicemesh/flavors/netmesh"
)

// netmesh main entry point.
func main() {
	// netmesh is a CN-infra based agent.
	agentVar := netmesh.NewAgent()
	if err := core.EventLoopWithInterrupt(agentVar, nil); err != nil {
		fmt.Printf("Failed to start core agent in network mesh service err: %v\n", err)
		os.Exit(1)
	}
}
