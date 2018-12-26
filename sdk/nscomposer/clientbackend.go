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

package nscomposer

import (
	"context"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
)

type ClientBackend interface {
	New() error
	Connect(ctx context.Context, connection *connection.Connection) error
	Close(ctx context.Context, connection *connection.Connection) error
}

type dummyClientBackend struct{}

func (*dummyClientBackend) New() error { return nil }
func (*dummyClientBackend) Connect(ctx context.Context, connection *connection.Connection) error {
	return nil
}
func (*dummyClientBackend) Close(ctx context.Context, connection *connection.Connection) error {
	return nil
}
