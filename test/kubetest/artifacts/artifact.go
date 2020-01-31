// Copyright (c) 2019-2020 Cisco Systems, Inc.
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

//Package artifacts provides API for manage tests artifacts.
package artifacts

// Artifact represents specific content
type Artifact interface {
	Name() string
	Kind() string
	Content() []byte
}

//New creates new artifact with specific name, kind and content
func New(name, kind string, content []byte) Artifact {
	return &artifact{
		name:    name,
		kind:    kind,
		content: content,
	}
}

type artifact struct {
	name    string
	kind    string
	content []byte
}

func (a *artifact) Name() string {
	return a.name
}

func (a *artifact) Kind() string {
	return a.kind
}

func (a *artifact) Content() []byte {
	return a.content
}
