// Copyright (c) 2019 Cisco Systems, Inc.
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

package utils

//Executor synchronously exectues functions
type Executor interface {
	//AsyncExec pushes function into the queue and not wait for function completed
	AsyncExec(func())
	//AsyncExec pushes function into the queue and wait for function completed
	SyncExec(func())
}

// NewExecutor creates new executor with specific queue length
func NewExecutor(maxQueueLen int) Executor {
	if maxQueueLen < 0 {
		panic("maxQueueLen should be more ore equals 0")
	}
	ex := &executor{
		executables: make(chan operation, maxQueueLen),
	}
	go ex.run()
	return ex
}

type executor struct {
	executables chan operation
}

type operation struct {
	f    func()
	done chan struct{}
}

func (e *executor) AsyncExec(f func()) {
	e.executables <- operation{f: f}
}

func (e *executor) SyncExec(f func()) {
	done := make(chan struct{})
	e.executables <- operation{f: f, done: done}
	<-done
}

func (e *executor) run() {
	for exec := range e.executables {
		exec.f()
		if exec.done != nil {
			close(exec.done)
		}
	}
}
