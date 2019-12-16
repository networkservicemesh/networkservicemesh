// Copyright (c) 2019 Cisco and/or its affiliates.
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

package tests

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/commands"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

func TestCloudTestAfterAllWorksCorrectly(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	provider := &config.ClusterProviderConfig{
		Timeout:    100,
		Name:       "provider",
		NodeCount:  1,
		Kind:       "shell",
		RetryCount: 1,
		Instances:  1,
		Scripts: map[string]string{
			"config":  "echo ./.tests/config",
			"start":   "echo started",
			"prepare": "echo prepared",
			"install": "echo installed",
			"stop":    "echo stopped",
		},
		Enabled: true,
	}
	testConfig.Providers = append(testConfig.Providers, provider)

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:     "test1",
		Timeout:  15,
		Kind:     "shell",
		Run:      "echo first",
		Env:      []string{"A=worked", "B=$(test-name)"},
		AfterAll: "echo ${B} ${A}",
	})
	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "test2",
		Timeout: 15,
		Kind:    "shell",
		Run:     "echo second",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})

	g.Expect(report).NotTo(BeNil())

	path := path.Join(tmpDir, provider.Name+"-1", "006-test2-run.log")
	content, err := ioutil.ReadFile(path)
	g.Expect(err).Should(BeNil())
	g.Expect(string(content)).Should(ContainSubstring("AfterAll worked"))
}
