package tests

import (
	"io/ioutil"
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
)

func TestClusterConfiguration(t *testing.T) {
	g := NewWithT(t)

	var testConfig config.CloudTestConfig

	file1, err := ioutil.ReadFile("./config1.yaml")
	g.Expect(err).To(BeNil())

	err = yaml.Unmarshal(file1, &testConfig)
	g.Expect(err).To(BeNil())
	g.Expect(len(testConfig.Providers)).To(Equal(3))
	g.Expect(testConfig.Reporting.JUnitReportFile).To(Equal("./.tests/junit.xml"))

}
