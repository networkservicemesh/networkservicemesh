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

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/jaeger"
	"github.com/networkservicemesh/networkservicemesh/utils"

	v1 "k8s.io/api/core/v1"
)

const (
	containerRepoEnv                   = "CONTAINER_REPO"
	containerTagEnv                    = "CONTAINER_TAG"
	containerTagDefault                = "master"
	containerForcePullEnv              = "CONTAINER_FORCE_PULL"
	jaegerVersionEnv      utils.EnvVar = "JAEGER_IMAGE_VERSION"
	containerRepoDefault               = "networkservicemesh"
	defaultJaegerVersion               = "1.14.0"
)

var containerRepo = ""
var containerTag = containerTagDefault
var containerForcePull = false
var jaegerVersion = ""

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
	jaegerVersion = jaegerVersionEnv.GetStringOrDefault(defaultJaegerVersion)
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

	if utils.EnvVar(tools.InsecureEnv).GetBooleanOrDefault(false) {
		c.Env = append(c.Env, v1.EnvVar{Name: tools.InsecureEnv, Value: "true"})
	}

	// Update Jaeger
	if utils.EnvVar("TRACER_ENABLED").GetBooleanOrDefault(true) {
		logrus.Infof("Added jaeger env for container %v", c.Name)
		c.Env = append(c.Env, jaeger.Env()...)
	}

	return *c
}
