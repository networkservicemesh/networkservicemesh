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

// Package core manages the lifecycle of all plugins (start, graceful
// shutdown) and defines the core lifecycle SPI. The core lifecycle SPI
// must be implemented by each plugin.

package idempotent_test

import (
	"errors"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	. "github.com/onsi/gomega"
	"testing"
)

type Plugin struct {
	idempotent.Impl
	RefCount int
	InitErr  error
	CloseErr error
}

func (plugin *Plugin) Init() error {
	return plugin.IdempotentInit(plugin.init)
}

func (plugin *Plugin) init() error {
	plugin.RefCount++
	return plugin.InitErr
}

func (plugin *Plugin) Close() error {
	return plugin.IdempotentClose(plugin.close)
}

func (plugin *Plugin) close() error {
	plugin.RefCount--
	return plugin.CloseErr
}

func testIdempotentImpl(t *testing.T, expectedInitErr error, expectedCloseErr error) {
	RegisterTestingT(t)
	p := &Plugin{
		InitErr:  expectedInitErr,
		CloseErr: expectedCloseErr,
	}
	expectedInitErrMatcher := BeNil()
	if expectedInitErr != nil {
		expectedInitErrMatcher = Equal(expectedInitErr)
	}
	expectedCloseErrorMatcher := BeNil()
	if expectedCloseErr != nil {
		expectedCloseErrorMatcher = Equal(expectedCloseErr)
	}

	_, ok := interface{}(p).(idempotent.Interface)
	Expect(ok).To(BeTrue())
	Expect(p.IsClosed()).To(BeFalse())
	Expect(p.IsIdempotent()).To(BeTrue())

	// Init
	err := p.Init()
	Expect(err).To(expectedInitErrMatcher)
	Expect(p.RefCount).To(Equal(1)) // See p.init() was called

	// Init again
	err = p.Init()
	Expect(err).To(expectedInitErrMatcher) // See the correct error, even though p.init() wasn't called again
	Expect(p.RefCount).To(Equal(1))        // See plugin.init() wasn't called again

	// Close
	err = p.Close()
	Expect(err).To(BeNil())         // See a nil error, because p.close() wasn't called
	Expect(p.RefCount).To(Equal(1)) // See plugin.close() wasn't called
	Expect(p.IsClosed()).To(BeFalse())

	// Close again
	err = p.Close()
	Expect(err).To(expectedCloseErrorMatcher) // See the correct close error
	Expect(p.RefCount).To(Equal(0))           // See p.close() was called
	Expect(p.IsClosed()).To(BeTrue())

	// Close even though p.close() was already been called
	err = p.Close()
	Expect(err).To(expectedCloseErrorMatcher) // See the correct close error, even though p.close() wasn't called again
	Expect(p.RefCount).To(Equal(0))           // See p.close() was called
	Expect(p.IsClosed()).To(BeTrue())

	// Try to re-init after true Close()
	err = p.Init()
	Expect(err).ToNot(BeNil())
	Expect(err.Error()).To(Equal(idempotent.ReinitErrorStr)) // Confirm we get a ReinitErrorStr
	Expect(p.RefCount).To(Equal(0))                          // See plugin.init() wasn't called again
	Expect(p.IsClosed()).To(BeTrue())

}

func TestIdemPotentImpl(t *testing.T) {
	testIdempotentImpl(t, nil, nil)
}

func TestIdemPotentImplInitError(t *testing.T) {
	testIdempotentImpl(t, errors.New("init error"), nil)
}

func TestIdemPotentImplCloseError(t *testing.T) {
	testIdempotentImpl(t, nil, errors.New("close error"))
}
