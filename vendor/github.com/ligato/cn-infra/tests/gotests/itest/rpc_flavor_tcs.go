package itest

import (
	"testing"

	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/flavors/rpc"
	"github.com/ligato/cn-infra/rpc/rest"
	"github.com/ligato/cn-infra/rpc/rest/mock"
	"github.com/onsi/gomega"
)

type suiteFlavorRPC struct {
	T *testing.T
	AgentT
	Given
	When
	Then
}

// Setup registers gomega and starts the agent with the flavor argument
func (t *suiteFlavorRPC) Setup(flavor core.Flavor, golangT *testing.T) {
	t.AgentT.Setup(flavor, t.t)
}

// MockFlavorRPC initializes RPC flavor with HTTP mock
func MockFlavorRPC() (*rpc.FlavorRPC, *mock.HTTPMock) {
	httpMock := &mock.HTTPMock{}
	return &rpc.FlavorRPC{
		HTTP: *rest.FromExistingServer(httpMock.SetHandler),
	}, httpMock
}

// TC01 asserts that injection works fine and agent starts & stops
func (t *suiteFlavorRPC) TC01StartStop() {
	flavor, _ := MockFlavorRPC()
	t.Setup(flavor, t.T)
	defer t.Teardown()

	gomega.Expect(t.agent).ShouldNot(gomega.BeNil(), "agent is not initialized")
}

/*TODO TC03 check that status check in flavor works
func (t *suiteFlavorRPC) TC03StatusCheck() {
	flavor, httpMock := MockFlavorRPC()
	t.Setup(flavor, t.T)
	defer t.Teardown()

	tstPlugin := core.PluginName("tstPlugin")
	flavor.StatusCheck.Register(tstPlugin, nil)
	flavor.StatusCheck.ReportStateChange(tstPlugin, "tst", nil)

	result, err := httpMock.NewRequest("GET", flavor.ServiceLabel.GetAgentPrefix()+
		"/check/status/v1/agent", nil)
	gomega.Expect(err).Should(gomega.BeNil(), "logger is not initialized")
	gomega.Expect(result).ShouldNot(gomega.BeNil(), "http result is not initialized")
	gomega.Expect(result.StatusCode).
		Should(gomega.BeEquivalentTo(200), "status code")
}
*/
