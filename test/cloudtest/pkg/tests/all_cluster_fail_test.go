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

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	failedP := createProvider(testConfig, "b_provider")
	failedP.Scripts["start"] = "echo starting\nexit 2"

	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 6"))

	g.Expect(report).NotTo(BeNil())

	rootSuite := report.Suites[0]

	g.Expect(len(rootSuite.Suites)).To(Equal(2))

	g.Expect(len(rootSuite.Suites[0].Suites)).To(Equal(2))

	g.Expect(rootSuite.Suites[0].Failures).To(Equal(4))
	g.Expect(rootSuite.Suites[0].Tests).To(Equal(6))
	g.Expect(len(rootSuite.Suites[0].Suites[0].TestCases)).To(Equal(3))
	g.Expect(len(rootSuite.Suites[0].Suites[1].TestCases)).To(Equal(3))

	g.Expect(rootSuite.Suites[1].Failures).To(Equal(2))
	g.Expect(rootSuite.Suites[1].Tests).To(Equal(2))
	g.Expect(len(rootSuite.Suites[1].TestCases)).To(Equal(2))

	// Do assertions
}

func TestClusterInstancesOnFailGoRunner(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	failedP := createProvider(testConfig, "b_provider")
	failedP.Scripts["start"] = "echo starting\nexit 2"

	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
		OnFail:      `echo >>>Running on fail script<<<`,
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 6"))

	g.Expect(report).NotTo(BeNil())

	rootSuite := report.Suites[0]

	g.Expect(len(rootSuite.Suites)).To(Equal(2))

	g.Expect(rootSuite.Suites[0].Failures).To(Equal(4))
	g.Expect(rootSuite.Suites[0].Tests).To(Equal(6))
	g.Expect(len(rootSuite.Suites[0].Suites[0].TestCases)).To(Equal(3))
	g.Expect(len(rootSuite.Suites[0].Suites[1].TestCases)).To(Equal(3))

	g.Expect(rootSuite.Suites[1].Failures).To(Equal(2))
	g.Expect(rootSuite.Suites[1].Tests).To(Equal(2))
	g.Expect(len(rootSuite.Suites[1].TestCases)).To(Equal(2))

	foundFailTest := false

	for _, execSuite := range rootSuite.Suites[0].Suites {
		if execSuite.Name == "a_provider" {
			for _, testCase := range execSuite.TestCases {
				if testCase.Name == "TestFail" {
					g.Expect(testCase.Failure).NotTo(BeNil())
					g.Expect(strings.Contains(testCase.Failure.Contents, ">>>Running on fail script<<<")).To(Equal(true))
					foundFailTest = true
				} else {
					g.Expect(testCase.Failure).Should(BeNil())
				}
			}
		}
	}
	g.Expect(foundFailTest).Should(BeTrue())
}

func TestClusterInstancesOnFailShellRunner(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:    "pass",
		Timeout: 15,
		Kind:    "shell",
		Run:     "echo pass",
		OnFail:  `echo >>>Running on fail script<<<`,
	})
	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
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

	for _, executionSuite := range report.Suites[0].Suites {
		testCase := executionSuite.Suites[0].TestCases[0]
		if executionSuite.Name == "fail" {
			g.Expect(testCase.Failure).NotTo(BeNil())
			g.Expect(strings.Contains(testCase.Failure.Contents, ">>>Running on fail script<<<")).To(Equal(true))
			foundFailTest = true
		} else {
			g.Expect(testCase.Failure).Should(BeNil())
		}
	}
	g.Expect(foundFailTest).Should(BeTrue())
}

func TestClusterInstancesOnFailShellRunnerInterdomain(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	ap := createProvider(testConfig, "a_provider")
	ap.Scripts["config"] = "echo ./.tests/config.a"
	bp := createProvider(testConfig, "b_provider")
	bp.Scripts["config"] = "echo ./.tests/config.b"
	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:            "pass",
		Timeout:         15,
		ClusterCount:    2,
		ClusterSelector: []string{"a_provider", "b_provider"},
		Kind:            "shell",
		Run:             "echo pass",
		OnFail:          `echo >>>Running on fail script with ${KUBECONFIG} <<<`,
	})
	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:            "fail",
		Timeout:         15,
		ClusterCount:    2,
		ClusterSelector: []string{"a_provider", "b_provider"},
		Kind:            "shell",
		Run:             "make_all_happy()",
		OnFail:          `echo >>>Running on fail script with ${KUBECONFIG} <<<`,
	})
	testConfig.Reporting.JUnitReportFile = JunitReport

	logKeeper := utils.NewLogKeeper()
	defer logKeeper.Stop()

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 1"))
	foundFailTest := false

	for _, suite := range report.Suites[0].Suites {
		t := suite.Suites[0].TestCases[0]
		if suite.Name == "fail" {
			g.Expect(t.Failure).NotTo(Equal(BeNil()))
			g.Expect(strings.Contains(t.Failure.Contents, ">>>Running on fail script with ./.tests/config.a <<<")).To(Equal(true))
			g.Expect(strings.Contains(t.Failure.Contents, ">>>Running on fail script with ./.tests/config.b <<<")).To(Equal(true))
			foundFailTest = true
		} else {
			g.Expect(t.Failure).Should(BeNil())
		}
	}
	g.Expect(foundFailTest).Should(BeTrue())

	g.Expect(logKeeper.MessageCount("OnFail: running on fail script operations with")).To(Equal(2))
}
