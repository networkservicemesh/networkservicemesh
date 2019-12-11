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

package kubetest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/jaeger"
)

const (
	jaegerAPI      = "api"
	jaegerServices = "services"
	jaegerTraces   = "traces"
)

type jaegerAPIClient struct {
	client        http.Client
	apiServerPort int
}

//GetJaegerTraces rerturns map of service and traces
func GetJaegerTraces(k8s *K8s, jaegerPod *v1.Pod) map[string]string {
	fwd, err := k8s.NewPortForwarder(jaegerPod, jaeger.GetJaegerRestAPIPort())
	k8s.g.Expect(err).To(gomega.BeNil())
	err = fwd.Start()
	k8s.g.Expect(err).To(gomega.BeNil())
	defer fwd.Stop()
	result := map[string]string{}
	j := &jaegerAPIClient{
		apiServerPort: fwd.ListenPort,
	}
	logrus.Info(fwd.ListenPort)
	services := j.getServices()
	for _, s := range services {
		result[s] = j.getTracesByService(s)
	}
	return result
}

func (j *jaegerAPIClient) getTracesByService(service string) string {
	url := fmt.Sprintf("%v?service=%v", urlToLocalHost(j.apiServerPort, jaegerAPI, jaegerTraces), service)
	resp, err := j.client.Get(url)
	if err != nil {
		logrus.Errorf("An error during get jaeger traces from API: %v", err)
		return ""
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("An error during read jaeger traces response: %v", err)
		return ""
	}
	return string(bytes)
}

func (j *jaegerAPIClient) getServices() []string {
	resp, err := j.client.Get(urlToLocalHost(j.apiServerPort, jaegerAPI, jaegerServices))
	if err != nil {
		logrus.Errorf("An error during get jaeger services from API: %v", err)
		return nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	jsonObject := map[string]interface{}{}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("An error during read jaeger services response: %v", err)
		return nil
	}
	err = json.Unmarshal([]byte(strings.ReplaceAll(string(bytes), "\\\"", "\"")), &jsonObject)
	if err != nil {
		logrus.Errorf("An error during unmarshal jaeger services response: %v", err)
		return nil
	}

	if v, ok := jsonObject["data"].([]interface{}); ok {
		result := []string{}
		for _, item := range v {
			result = append(result, fmt.Sprint(item))
		}
		logrus.Info(v)
		return result
	}

	return nil
}

func urlToLocalHost(port int, parts ...string) string {
	u, _ := url.Parse(fmt.Sprintf("http://0.0.0.0:%v", port))
	fullPath := append([]string{}, u.Path)
	fullPath = append(fullPath, parts...)
	u.Path = path.Join(fullPath...)
	return u.String()
}
