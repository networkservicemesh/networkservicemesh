package tests

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/commands"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

func TestClusterInstancesFailed(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	failedP := createProvider(testConfig, "b_provider")
	failedP.Scripts["start"] = "echo starting\nexit 2"

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 3"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(2))
	g.Expect(report.Suites[0].Failures).To(Equal(1))
	g.Expect(report.Suites[0].Tests).To(Equal(3))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(3))

	g.Expect(report.Suites[1].Failures).To(Equal(2))
	g.Expect(report.Suites[1].Tests).To(Equal(5))
	g.Expect(len(report.Suites[1].TestCases)).To(Equal(5))

	// Do assertions
}

func TestClusterInstancesFailedSpecificTestList(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	failedP := createProvider(testConfig, "b_provider")
	failedP.Scripts["start"] = "echo starting\nexit 2"

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
		Source: config.ExecutionSource{
			Tests: []string{"TestPass", "TestTimeout", "TestFail"},
		},
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 3"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(2))
	g.Expect(report.Suites[0].Failures).To(Equal(1))
	g.Expect(report.Suites[0].Tests).To(Equal(3))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(3))

	g.Expect(report.Suites[1].Failures).To(Equal(2))
	g.Expect(report.Suites[1].Tests).To(Equal(5))
	g.Expect(len(report.Suites[1].TestCases)).To(Equal(5))

	// Do assertions
}

func TestClusterInstancesOnFailGoRunner(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	failedP := createProvider(testConfig, "b_provider")
	failedP.Scripts["start"] = "echo starting\nexit 2"

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
		OnFail:      `echo >>>Running on fail script<<<`,
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 3"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(2))
	g.Expect(report.Suites[0].Failures).To(Equal(1))
	g.Expect(report.Suites[0].Tests).To(Equal(3))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(3))

	g.Expect(report.Suites[1].Failures).To(Equal(2))
	g.Expect(report.Suites[1].Tests).To(Equal(5))
	g.Expect(len(report.Suites[1].TestCases)).To(Equal(5))

	foundFailTest := false

	for _, t := range report.Suites[0].TestCases {
		if t.Name == "_TestFail" {
			g.Expect(t.Failure).NotTo(Equal(BeNil()))
			g.Expect(strings.Contains(t.Failure.Contents, ">>>Running on fail script<<<")).To(Equal(true))
			foundFailTest = true
		} else {
			g.Expect(t.Failure).Should(BeNil())
		}
	}
	g.Expect(foundFailTest).Should(BeTrue())
}

func TestClusterInstancesOnFailShellRunner(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "pass",
		Timeout: 15,
		Kind:    "shell",
		Run:     "echo pass",
		OnFail:  `echo >>>Running on fail script<<<`,
	})
	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "fail",
		Timeout: 15,
		Kind:    "shell",
		Run:     "make_all_happy()",
		OnFail:  `echo >>>Running on fail script<<<`,
	})
	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 1"))
	foundFailTest := false

	for _, t := range report.Suites[0].TestCases {
		if t.Name == "_fail" {
			g.Expect(t.Failure).NotTo(Equal(BeNil()))
			g.Expect(strings.Contains(t.Failure.Contents, ">>>Running on fail script<<<")).To(Equal(true))
			foundFailTest = true
		} else {
			g.Expect(t.Failure).Should(BeNil())
		}
	}
	g.Expect(foundFailTest).Should(BeTrue())
}
