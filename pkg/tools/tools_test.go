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

// +build unit_test

package tools_test

import (
	"reflect"
	"testing"

	. "github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

func TestParseKVStringToMap(t *testing.T) {
	type args struct {
		input string
		sep   string
		kvsep string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "SimpleConfig",
			args: args{
				input: "nsm1:icmp-responder-nse, eth12:vpngateway",
				sep:   ",",
				kvsep: ":",
			},
			want: map[string]string{
				"nsm1":  "icmp-responder-nse",
				"eth12": "vpngateway",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseKVStringToMap(tt.args.input, tt.args.sep, tt.args.kvsep); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseKVStringToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
