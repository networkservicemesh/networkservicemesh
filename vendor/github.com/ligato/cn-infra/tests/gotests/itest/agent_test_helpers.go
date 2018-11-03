package itest

import (
	"testing"

	"github.com/ligato/cn-infra/core"
	"github.com/onsi/gomega"
)

// AgentT is similar to what testing.T is in golang packages.
type AgentT struct {
	agent *core.Agent
	t     *testing.T
}

// Given is composition of multiple test step methods (see BDD Given keyword)
type Given struct {
}

// When is composition of multiple test step methods (see BDD When keyword)
type When struct {
}

// Then is composition of multiple test step methods (see BDD Then keyword)
type Then struct {
}

// Setup registers gomega and starts the agent with the flavor argument
func (t *AgentT) Setup(flavor core.Flavor, golangT *testing.T) {
	gomega.RegisterTestingT(golangT)
	t.t = golangT

	t.agent = core.NewAgent(flavor)
	err := t.agent.Start()
	if err != nil {
		golangT.Fatal("error starting agent ", err)
	}
}

// Teardown stops the agent
func (t *AgentT) Teardown() {
	if t.agent != nil {
		err := t.agent.Stop()
		if err != nil {
			t.t.Fatal("error stoppig agent ", err)
		}
	}
}

/*/ KafkaMock allows to flavors the Kafka Mock
func (g *GivenKW) KafkaMock(setupVppMock func(adapter *mock.VppAdapter)) *GivenAndKW {
	g.testOpts.KafkaMock = kafkamux.NewMultiplexerMock(g.testing)

	return &GivenAndKW{g}
}*/
