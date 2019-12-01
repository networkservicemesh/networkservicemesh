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

	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:     "test1",
		Timeout:  15,
		Kind:     "shell",
		Run:      "echo first",
		Env:      []string{"A=worked", "B=$(test-name)"},
		AfterAll: "echo ${B} ${A}",
	})
	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:    "test2",
		Timeout: 15,
		Kind:    "shell",
		Run:     "echo second",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})

	g.Expect(report).NotTo(BeNil())

	path := path.Join(tmpDir, provider.Name+"-1", "007-test2-run.log")
	content, err := ioutil.ReadFile(path)
	g.Expect(err).Should(BeNil())
	g.Expect(string(content)).Should(ContainSubstring("AfterAll worked"))
}
