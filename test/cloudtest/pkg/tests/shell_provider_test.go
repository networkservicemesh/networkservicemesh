package tests

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/commands"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

const (
	JunitReport = "reporting/junit.xml"
)

type testValidationFactory struct {
}

type testValidator struct {
	location string
	config   *config.ClusterProviderConfig
}

func (v *testValidator) WaitValid(context context.Context) error {
	return nil
}

func (v *testValidator) Validate() error {
	// Validation is passed for now
	return nil
}

func (*testValidationFactory) CreateValidator(config *config.ClusterProviderConfig, location string) (k8s.KubernetesValidator, error) {
	return &testValidator{
		config:   config,
		location: location,
	}, nil
}

func TestShellProvider(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	createProvider(testConfig, "b_provider")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple_tagged",
		Timeout: 15,
		Source: config.ExecutionSource{
			Tags: []string{"basic"},
		},
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 4"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(2))
	g.Expect(report.Suites[0].Failures).To(Equal(2))
	g.Expect(report.Suites[0].Tests).To(Equal(6))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(6))

	// Do assertions
}

func createProvider(testConfig *config.CloudTestConfig, name string) *config.ClusterProviderConfig {
	provider := &config.ClusterProviderConfig{
		Timeout:    100,
		Name:       name,
		NodeCount:  1,
		Kind:       "shell",
		RetryCount: 1,
		Instances:  2,
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
	return provider
}

func TestInvalidProvider(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	delete(testConfig.Providers[0].Scripts, "start")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     2,
		PackageRoot: "./sample",
	})

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	logrus.Error(err.Error())
	g.Expect(err.Error()).To(Equal("Failed to create cluster instance. Error invalid start script"))

	g.Expect(report).To(BeNil())
	// Do assertions
}

func TestRequireEnvVars(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir

	createProvider(testConfig, "a_provider")

	testConfig.Providers[0].EnvCheck = append(testConfig.Providers[0].EnvCheck, []string{
		"KUBECONFIG", "QWE",
	}...)

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     2,
		PackageRoot: "./sample",
	})

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	logrus.Error(err.Error())
	g.Expect(err.Error()).To(Equal(
		"Failed to create cluster instance. Error environment variable are not specified  Required variables: [KUBECONFIG QWE]"))

	g.Expect(report).To(BeNil())
	// Do assertions
}

func TestRequireEnvVars_DEPS(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir

	createProvider(testConfig, "a_provider")

	testConfig.Providers[0].EnvCheck = append(testConfig.Providers[0].EnvCheck, "PACKET_AUTH_TOKEN")
	testConfig.Providers[0].EnvCheck = append(testConfig.Providers[0].EnvCheck, "PACKET_PROJECT_ID")

	_ = os.Setenv("PACKET_AUTH_TOKEN", "token")
	_ = os.Setenv("PACKET_PROJECT_ID", "id")

	testConfig.Providers[0].Env = append(testConfig.Providers[0].Env, []string{
		"CLUSTER_RULES_PREFIX=packet",
		"CLUSTER_NAME=$(cluster-name)-$(uuid)",
		"KUBECONFIG=$(tempdir)/config",
		"TERRAFORM_ROOT=$(tempdir)/terraform",
		"TF_VAR_auth_token=${PACKET_AUTH_TOKEN}",
		"TF_VAR_master_hostname=devci-${CLUSTER_NAME}-master",
		"TF_VAR_worker1_hostname=ci-${CLUSTER_NAME}-worker1",
		"TF_VAR_project_id=${PACKET_PROJECT_ID}",
		"TF_VAR_public_key=${TERRAFORM_ROOT}/sshkey.pub",
		"TF_VAR_public_key_name=key-${CLUSTER_NAME}",
		"TF_LOG=DEBUG",
	}...)

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     2,
		PackageRoot: "./sample",
	})

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 2"))

	g.Expect(report).ToNot(BeNil())
	// Do assertions
}

func TestShellProviderShellTest(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	createProvider(testConfig, "b_provider")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple_shell",
		Timeout: 150000,
		Kind:    "shell",
		Run: strings.Join([]string{
			"pwd",
			"ls -la",
			"echo $KUBECONFIG",
		}, "\n"),
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple_shell_fail",
		Timeout: 15,
		Kind:    "shell",
		Run: strings.Join([]string{
			"pwd",
			"ls -la",
			"exit 1",
		}, "\n"),
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 4"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(2))
	g.Expect(report.Suites[0].Failures).To(Equal(2))
	g.Expect(report.Suites[0].Tests).To(Equal(5))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(5))

	// Do assertions
}

func TestUnusedClusterShutdownByMonitor(t *testing.T) {
	g := NewWithT(t)
	logKeeper := utils.NewLogKeeper()
	defer logKeeper.Stop()
	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	p2 := createProvider(testConfig, "b_provider")
	p2.TestDelay = 7

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:            "simple",
		Timeout:         15,
		PackageRoot:     "./sample",
		ClusterSelector: []string{"a_provider"},
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple2",
		Timeout: 15,
		Source: config.ExecutionSource{
			Tags: []string{"basic"},
		},
		PackageRoot:     "./sample",
		ClusterSelector: []string{"b_provider"},
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 2"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(2))
	g.Expect(report.Suites[0].Failures).To(Equal(1))
	g.Expect(report.Suites[0].Tests).To(Equal(3))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(3))

	logKeeper.CheckMessagesOrder(t, []string{
		"All tasks for cluster group a_provider are complete. Starting cluster shutdown",
		"Destroying cluster  a_provider-",
		"Finished test execution",
	})
}

func TestMultiClusterTest(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	p1 := createProvider(testConfig, "a_provider")
	p2 := createProvider(testConfig, "b_provider")
	p3 := createProvider(testConfig, "c_provider")
	p4 := createProvider(testConfig, "d_provider")

	p1.Instances = 1
	p2.Instances = 1
	p3.Instances = 1
	p4.Instances = 1

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:            "simple",
		Timeout:         15,
		PackageRoot:     "./sample",
		ClusterSelector: []string{"a_provider"},
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple2",
		Timeout: 15,
		Source: config.ExecutionSource{
			Tags: []string{"interdomain"},
		},
		PackageRoot:     "./sample",
		ClusterCount:    2,
		KubernetesEnv:   []string{"CFG1", "CFG2"},
		ClusterSelector: []string{"a_provider", "b_provider"},
	})
	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple3",
		Timeout: 15,
		Source: config.ExecutionSource{
			Tags: []string{"interdomain"},
		},
		PackageRoot:     "./sample",
		ClusterCount:    2,
		KubernetesEnv:   []string{"CFG1", "CFG2"},
		ClusterSelector: []string{"c_provider", "d_provider"},
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 3"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(4))
	g.Expect(report.Suites[0].Failures).To(Equal(2))
	g.Expect(report.Suites[0].Tests).To(Equal(6))
	g.Expect(report.Suites[1].Tests).To(Equal(0))
	g.Expect(report.Suites[2].Tests).To(Equal(3))
	g.Expect(report.Suites[3].Tests).To(Equal(0))

	// Do assertions
}

func TestGlobalTimeout(t *testing.T) {
	g := NewWithT(t)

	testConfig := &config.CloudTestConfig{}
	testConfig.Timeout = 3

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &testValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("global timeout elapsed: 3 seconds"))

	g.Expect(report).NotTo(BeNil())

	g.Expect(len(report.Suites)).To(Equal(1))
	g.Expect(report.Suites[0].Failures).To(Equal(1))
	g.Expect(report.Suites[0].Tests).To(Equal(3))
	g.Expect(len(report.Suites[0].TestCases)).To(Equal(3))

	// Do assertions
}
