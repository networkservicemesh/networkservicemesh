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

	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
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
