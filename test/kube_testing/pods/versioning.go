// Copyright 2019 VMware, Inc.
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

package pods

import (
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
)

const (
	containerRegistryEnv  = "CONTAINER_REGISTRY"
	containerTagEnv       = "CONTAINER_TAG"
	containerTagDefault   = "latest"
	containerForcePullEnv = "CONTAINER_FORCE_PULL"
)

var containerRegistry = ""
var containerTag = "latest"
var containerForcePull = false

func init() {
	found := false
	containerRegistry, found = os.LookupEnv(containerRegistryEnv)

	containerTag, found = os.LookupEnv(containerTagEnv)
	if !found {
		containerTag = containerTagDefault
	}

	pull := os.Getenv(containerForcePullEnv)
	containerForcePull = (pull == "true")
}

func containerMod(c *v1.Container) v1.Container {
	c.Image = strings.Split(c.Image, ":")[0] + ":" + containerTag
	if len(containerRegistry) > 0 {
		c.Image = containerRegistry + "/" + c.Image
	}

	if containerForcePull {
		c.ImagePullPolicy = v1.PullAlways
	}
	return *c
}
