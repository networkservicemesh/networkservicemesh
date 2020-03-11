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

package kubetest

// ClearOption means how test will prepare and cleanup resources
type ClearOption int32

const (
	// NoClear means that test should not do prepare.
	NoClear ClearOption = iota
	// DefaultClear means that test should clear resources on prepare and on cleanup.
	DefaultClear
	// ReuseNSMResources means that test can try to reuse exists service accounts and nsm / forwarder pods.
	ReuseNSMResources
)
