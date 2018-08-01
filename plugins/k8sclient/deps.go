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

package k8sclient

import (
	"github.com/ligato/networkservicemesh/plugins/logger"
)

// Deps defines dependencies of k8sclient plugin.
type Deps struct {
	Name string
	Log  logger.FieldLoggerPlugin
	// Kubeconfig with k8s cluster address and access credentials to use.
	KubeConfig string `empty_value_ok:"true"`
}
