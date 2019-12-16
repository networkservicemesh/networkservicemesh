// Copyright (c) 2019 Cisco Systems, Inc and/or its affiliates.
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

// Package tools - helpful multi module tools
package tools

import (
	. "context"

	. "google.golang.org/grpc/metadata"
)

// MetadataWithIncomingContext - Creates new context with incoming md attached.
func MetadataWithIncomingContext(srcCtx, incomingCtx Context) Context {
	if incomingMd, ok := FromIncomingContext(incomingCtx); ok {
		md := incomingMd
		if srcMd, ok := FromOutgoingContext(srcCtx); ok {
			md = Join(md, srcMd)
		}
		return NewOutgoingContext(srcCtx, md)
	}
	return srcCtx
}

// MetadataWithPair - Creates new context with joined single MD (joined multiple values by a key)
func MetadataWithPair(srcCtx Context, kv ...string) Context {
	md := Pairs(kv...)
	if srcMd, ok := FromOutgoingContext(srcCtx); ok {
		md = Join(md, srcMd)
	}
	return NewOutgoingContext(srcCtx, md)
}

// MetadataFromIncomingContext - obtains the values from context for a given key.
func MetadataFromIncomingContext(ctx Context, key string) []string {
	if md, ok := FromIncomingContext(ctx); ok {
		return md.Get(key)
	}
	return []string{}
}
