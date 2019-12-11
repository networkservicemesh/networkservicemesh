// Copyright 2018-2019 Cisco Systems, Inc.
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

package connection

import (
	proto "github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// Clone clones connection
func (p *Path) Clone() *Path {
	return proto.Clone(p).(*Path)
}

func (p *Path) IsValid() error {
	if p == nil {
		return nil
	}
	if int(p.GetIndex()) >= len(p.GetPathSegments()) {
		return errors.New("Path.Index >= len(Path.PathSegments)")
	}
	return nil
}

func (p *Path) ExtendPath(id, name, requestToken string) *Path {
	if p == nil {
		return &Path{
			Index: 0,
			PathSegments: []*PathSegment{
				{
					RequestToken: requestToken,
				},
			},
		}
	}
	path := p.Clone()
	ps := path.GetPathSegments()[path.GetIndex()]
	ps.Id = id
	ps.Name = name
	path.PathSegments = append(path.PathSegments, &PathSegment{
		RequestToken: requestToken,
	})
	return path
}
