// Copyright (c) 2020 Cisco Systems, Inc.
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

package common

import "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"

// These functions are intentionally not put in api path_helper because they are particular to adapting the existing
// code to using Path and so intentionally overly simplified.

// Strings2Path is a utility function to create a Path from a list of NSMgr names
func Strings2Path(strs ...string) *connection.Path {
	return AppendStrings2Path(&connection.Path{}, strs...)
}

// AppendStrings2Path is a utility function to append PathSegments to a Path from a list of Names
func AppendStrings2Path(path *connection.Path, strs ...string) *connection.Path {
	for _, str := range strs {
		path.PathSegments = append(path.GetPathSegments(), &connection.PathSegment{Name: str})
	}
	return path
}
