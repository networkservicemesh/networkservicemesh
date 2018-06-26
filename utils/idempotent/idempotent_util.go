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

package idempotent

import "github.com/ligato/cn-infra/core"

func SafeInit(input interface{}) error {
	if input != nil {
		p, isPlugin := input.(core.Plugin)
		_, isIdempotent := input.(Interface)
		if isPlugin && isIdempotent {
			return p.Init()
		}
	}
	return nil
}

func SafeClose(input interface{}) error {
	if input != nil {
		p, isPlugin := input.(core.Plugin)
		_, isIdempotent := input.(Interface)
		if isPlugin && isIdempotent {
			return p.Close()
		}
	}
	return nil
}
