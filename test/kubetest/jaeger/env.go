// Copyright (c) 2019 Cisco Systems, Inc.
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

package jaeger

import (
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/utils"
)

const (
	prefix = "JAEGER"
	//RestAPIPort means port of ingester api server
	RestAPIPort utils.EnvVar = "JAEGER_REST_API_PORT"
	//AgentHost the hostname for communicating with agent via UDP
	AgentHost utils.EnvVar = "JAEGER_AGENT_HOST"
	//AgentPort the port for communicating with agent via UDP
	AgentPort utils.EnvVar = "JAEGER_AGENT_PORT"
	//UseService means use jaeger service. Useful for interdomain cases
	UseService utils.EnvVar = "USE_JAEGER_SERVICE"
	//NodePort means jaeger node port
	NodePort utils.EnvVar = "NODE_JAEGER_PORT"
)

//GetRestAPIPort returns jaeger API port
func GetRestAPIPort() int {
	return RestAPIPort.GetIntOrDefault(16686)
}

//GetNodePort returns jaeger node port
func GetNodePort() int32 {
	return int32(NodePort.GetIntOrDefault(30001))
}

//DefaultEnvValues returns default jaeger env values
func DefaultEnvValues() map[string]string {
	return map[string]string{
		"JAEGER_AGENT_HOST": "jaeger",
		"JAEGER_AGENT_PORT": "6831",
		"JAEGER_API_PORT":   "16686",
	}
}

//Env converts user's jaeger env to []v1.EnvVar
func Env() []v1.EnvVar {
	envs := os.Environ()
	envMap := map[string]string{}
	defaultEnvs := DefaultEnvValues()
	result := []v1.EnvVar{}
	for _, env := range envs {
		if strings.HasPrefix(env, prefix) {
			envParts := strings.Split(env, "=")
			if len(envParts) < 2 {
				continue
			}
			envMap[envParts[0]] = envParts[1]
		}
	}
	for k, v := range defaultEnvs {
		if envMap[k] == "" {
			envMap[k] = v
		}
	}
	for k, v := range envMap {
		result = append(result, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return result
}
