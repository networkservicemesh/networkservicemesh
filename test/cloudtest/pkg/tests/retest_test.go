// Copyright (c) 2019 Cisco Systems, Inc and/or its affiliates.
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

// Package tests - Cloud test tests
package tests

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/commands"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

func TestRestartRequest(t *testing.T) {
	g := NewWithT(t)

	logKeeper := utils.NewLogKeeper()
	defer logKeeper.Stop()

	testConfig := &config.CloudTestConfig{
		RetestConfig: config.RetestConfig{
			Patterns:     []string{"#Please_RETEST#"},
			RestartCount: 2,
		},
	}
	testConfig.Timeout = 3000

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name: "simple",
		Source: config.ExecutionSource{
			Tags: []string{"request_restart"},
		},
		Timeout:     1500,
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 1"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(1))
	g.Expect(report.Suites[0].Failures).To(Equal(1))
	g.Expect(report.Suites[0].Tests).To(Equal(2))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(2))

	logKeeper.CheckMessagesOrder(t, []string{
		"Starting TestRequestRestart",
		"Re schedule task TestRequestRestart reason: rerun-request",
		"Test TestRequestRestart retry count 2 exceed: err",
	})
	g.Expect(logKeeper.MessageCount("Re schedule task TestRequestRestart reason: rerun-request")).To(Equal(2))
}

func TestRestartRetestDestroyCluster(t *testing.T) {
	g := NewWithT(t)

	logKeeper := utils.NewLogKeeper()
	defer logKeeper.Stop()

	testConfig := &config.CloudTestConfig{
		RetestConfig: config.RetestConfig{
			Patterns:       []string{"#Please_RETEST#"},
			RestartCount:   3,
			AllowedRetests: 1,
			WarmupTimeout:  0,
		},
	}
	testConfig.Timeout = 1000

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	p := createProvider(testConfig, "a_provider")
	p.Instances = 1
	p.RetryCount = 1

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name: "simple",
		Source: config.ExecutionSource{
			Tags: []string{"request_restart"},
		},
		Timeout:     1500,
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 1"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(1))
	g.Expect(report.Suites[0].Failures).To(Equal(1))
	g.Expect(report.Suites[0].Tests).To(Equal(3))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(3))

	logKeeper.CheckMessagesOrder(t, []string{
		"Starting TestRequestRestart",
		"Reached a limit of re-tests per cluster instance",
		"Destroying cluster",
		"Re schedule task TestRequestRestart reason: rerun-request",
		"Starting cluster ",
		"Starting TestRequestRestart",
		"Re schedule task TestRequestRestart reason: rerun-request",
		"Move task to skipped since no clusters could execute it: TestRequestRestart",
	})
}

func TestRestartRequestRestartCluster(t *testing.T) {
	g := NewWithT(t)

	logKeeper := utils.NewLogKeeper()
	defer logKeeper.Stop()

	testConfig := &config.CloudTestConfig{
		RetestConfig: config.RetestConfig{
			Patterns:       []string{"#Please_RETEST#"},
			RestartCount:   3,
			AllowedRetests: 2,
			WarmupTimeout:  1,
		},
	}
	testConfig.Timeout = 1000

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	p := createProvider(testConfig, "a_provider")
	p.Instances = 1
	p.RetryCount = 10

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name: "simple",
		Source: config.ExecutionSource{
			Tags: []string{"request_restart"},
		},
		Timeout:     1500,
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 1"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(1))
	g.Expect(report.Suites[0].Failures).To(Equal(1))
	g.Expect(report.Suites[0].Tests).To(Equal(2))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(2))

	logKeeper.CheckMessagesOrder(t, []string{
		"Starting TestRequestRestart",
		"Reached a limit of re-tests per cluster instance",
		"Destroying cluster",
		"Starting cluster ",
		"Test TestRequestRestart retry count 3 exceed: err: failed to run go test",
	})
	g.Expect(logKeeper.MessageCount("Re schedule task TestRequestRestart reason: rerun-request")).To(Equal(3))
}

func TestRestartRequestSkip(t *testing.T) {
	g := NewWithT(t)

	logKeeper := utils.NewLogKeeper()
	defer logKeeper.Stop()

	testConfig := &config.CloudTestConfig{
		RetestConfig: config.RetestConfig{
			Patterns:         []string{"#Please_RETEST#"},
			RestartCount:     2,
			RetestFailResult: "skip",
		},
	}
	testConfig.Timeout = 3000

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name: "simple",
		Source: config.ExecutionSource{
			Tags: []string{"request_restart"},
		},
		Timeout:     1500,
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err).To(BeNil())

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(1))
	g.Expect(report.Suites[0].Failures).To(Equal(0))
	g.Expect(report.Suites[0].Tests).To(Equal(2))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(2))

	for _, tt := range report.Suites[0].TestCases {
		if tt.Name == "_TestRequestRestart" {
			g.Expect(tt.SkipMessage.Message).To(Equal("Test TestRequestRestart retry count 2 exceed: err: failed to run go test . -test.timeout 50m0s -count 1 --run \"^(TestRequestRestart)\\\\z\" --tags \"request_restart\" --test.v ExitCode: 1"))
		}
	}

	logKeeper.CheckMessagesOrder(t, []string{
		"Starting TestRequestRestart",
		"Re schedule task TestRequestRestart reason: rerun-request",
		"Test TestRequestRestart retry count 2 exceed: err",
		"Re schedule task TestRequestRestart reason: skipped",
	})
	g.Expect(logKeeper.MessageCount("Re schedule task TestRequestRestart reason: rerun-request")).To(Equal(2))
}
