// Copyright (c) 2019 Cisco and/or its affiliates.
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

package common

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/spanhelper"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

// SpanHelperFromConnection - construct new span helper with span from context is pressent or span from connection object
func SpanHelperFromConnection(ctx context.Context, clientConnection *model.ClientConnection, operation string) spanhelper.SpanHelper {
	if clientConnection != nil && clientConnection.Span != nil {
		return spanhelper.SpanHelperWithSpan(ctx, clientConnection.Span, operation)
	}
	return spanhelper.SpanHelperFromContext(ctx, operation)
}
