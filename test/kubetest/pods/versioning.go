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

	"github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
)

const (
	containerRepoEnv      = "CONTAINER_REPO"
	containerTagEnv       = "CONTAINER_TAG"
	containerTagDefault   = "latest"
	containerForcePullEnv = "CONTAINER_FORCE_PULL"
	containerRepoDefault  = "networkservicemesh"
)

var containerRepo = ""
var containerTag = "latest"
var containerForcePull = false

func init() {
	found := false
	containerRepo, found = os.LookupEnv(containerRepoEnv)

	if !found {
		containerRepo = containerRepoDefault
	}

	containerTag, found = os.LookupEnv(containerTagEnv)
	if !found {
		containerTag = containerTagDefault
	}

	pull := os.Getenv(containerForcePullEnv)
	containerForcePull = pull == "true"
}

func containerMod(c *v1.Container) v1.Container {
	if strings.HasPrefix(c.Image, containerRepoDefault) {
		c.Image = strings.Split(c.Image, ":")[0] + ":" + containerTag
		if len(containerRepo) > 0 {
			c.Image = strings.Replace(c.Image, containerRepoDefault, containerRepo, -1)
		}

		if containerForcePull {
			c.ImagePullPolicy = v1.PullAlways
		}
	}

	// Update Jaeger
	if os.Getenv("TRACER_ENABLED") == "true" {
		logrus.Infof("TRACER_ENABLED %v", c.Name)
		c.Env = append(c.Env, newJaegerEnvVar()...)
	}

	return *c
}
