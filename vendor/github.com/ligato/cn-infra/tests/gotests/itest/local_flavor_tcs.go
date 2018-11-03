package itest

import (
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/onsi/gomega"
	"testing"
)

type suiteFlavorLocal struct {
	T *testing.T
	AgentT
	Given
	When
	Then
}

// TC01 asserts that injection works fine and agent starts & stops
func (t *suiteFlavorLocal) TC01StartStop() {
	flavor := &local.FlavorLocal{}
	t.Setup(flavor, t.T)
	defer t.Teardown()

	gomega.Expect(t.agent).ShouldNot(gomega.BeNil(), "agent is not initialized")
}

// TC02 check that logger in flavor works
func (t *suiteFlavorLocal) TC02Logger() {
	flavor := &local.FlavorLocal{}
	t.Setup(flavor, t.T)
	defer t.Teardown()

	logger := flavor.LogRegistry().NewLogger("myTest")
	gomega.Expect(logger).ShouldNot(gomega.BeNil(), "logger is not initialized")
	logger.Debug("log msg")
}

// TC03 check that status check in flavor works
func (t *suiteFlavorLocal) TC03StatusCheck() {
	flavor := &local.FlavorLocal{}
	t.Setup(flavor, t.T)
	defer t.Teardown()

	tstPlugin := core.PluginName("tstPlugin")
	flavor.StatusCheck.Register(tstPlugin, nil)
	flavor.StatusCheck.ReportStateChange(tstPlugin, "tst", nil)

	//TODO assert flavor.StatusCheck using IDX map???
}
