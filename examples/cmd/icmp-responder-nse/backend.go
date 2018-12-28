// Copyright 2018 VMware, Inc.
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

package main

import (
	"context"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
)

type icmpResponderBackend struct {
}

func (ns *icmpResponderBackend) New() error {
	return nil
}

func (ns *icmpResponderBackend) Request(ctx context.Context, incoming, outgoing *connection.Connection, baseDir string) error {
	return nil
}

func (ns *icmpResponderBackend) Close(ctx context.Context, conn *connection.Connection, baseDir string) error {
	return nil
}
