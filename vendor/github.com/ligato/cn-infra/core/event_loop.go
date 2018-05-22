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

package core

import (
	"os"
	"os/signal"
	"syscall"
)

// EventLoopWithInterrupt starts an instance of the agent created with NewAgent().
// Agent is stopped when <closeChan> is closed, a user interrupt (SIGINT), or a
// terminate signal (SIGTERM) is received.
func EventLoopWithInterrupt(agent *Agent, closeChan chan struct{}) error {
	err := agent.Start()
	if err != nil {
		agent.Error("Error loading core: ", err)
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)
	select {
	case <-sigChan:
		agent.Println("Interrupt received, returning.")
	case <-closeChan:
	}

	err = agent.Stop()
	if err != nil {
		agent.Errorf("Agent stop error '%+v'", err)
	}
	return err
}
