package dataplane

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent/dataplane/state"
)

func TestBasicDataplaneChain(t *testing.T) {
	gomega.RegisterTestingT(t)
	assert := gomega.NewWithT(t)
	first := &testChainDataplaneServer{}
	second := &testChainDataplaneServer{}

	chain := ChainOf(first, second)
	chain.Request(context.Background(), nil)
	assert.Expect(first.requestCount).Should(gomega.Equal(1))
	assert.Expect(second.requestCount).Should(gomega.Equal(1))
	assert.Expect(first.closeCount).Should(gomega.Equal(0))
	assert.Expect(second.closeCount).Should(gomega.Equal(0))
	assert.Expect(first.monitorCount).Should(gomega.Equal(0))
	assert.Expect(second.monitorCount).Should(gomega.Equal(0))
	chain.Close(context.Background(), nil)
	assert.Expect(first.requestCount).Should(gomega.Equal(1))
	assert.Expect(second.requestCount).Should(gomega.Equal(1))
	assert.Expect(first.closeCount).Should(gomega.Equal(1))
	assert.Expect(second.closeCount).Should(gomega.Equal(1))
	assert.Expect(first.monitorCount).Should(gomega.Equal(0))
	assert.Expect(second.monitorCount).Should(gomega.Equal(0))
	chain.MonitorMechanisms(nil, nil)
	assert.Expect(first.requestCount).Should(gomega.Equal(1))
	assert.Expect(second.requestCount).Should(gomega.Equal(1))
	assert.Expect(first.closeCount).Should(gomega.Equal(1))
	assert.Expect(second.closeCount).Should(gomega.Equal(1))
	assert.Expect(first.monitorCount).Should(gomega.Equal(1))
	assert.Expect(second.monitorCount).Should(gomega.Equal(1))
}

func TestBranchDataplaneChain(t *testing.T) {
	gomega.RegisterTestingT(t)
	assert := gomega.NewWithT(t)
	first := &testChainDataplaneServer{}
	second := &branchChainDataplaneRequst{&testChainDataplaneServer{}}
	third := &testChainDataplaneServer{}
	chain := ChainOf(first, second, third)
	resp, err := chain.Request(context.Background(), nil)
	assert.Expect(resp).Should(gomega.BeNil())
	assert.Expect(err).Should(gomega.BeNil())
	assert.Expect(first.requestCount).Should(gomega.Equal(1))
	assert.Expect(second.requestCount).Should(gomega.Equal(1))
	assert.Expect(third.requestCount).Should(gomega.Equal(1))
	resp, err = chain.Request(context.Background(), nil)
	assert.Expect(resp).ShouldNot(gomega.BeNil())
	assert.Expect(err).Should(gomega.BeNil())
	assert.Expect(first.requestCount).Should(gomega.Equal(2))
	assert.Expect(second.requestCount).Should(gomega.Equal(2))
	assert.Expect(third.requestCount).Should(gomega.Equal(1))
	chain.Close(context.Background(), nil)
	assert.Expect(first.closeCount).Should(gomega.Equal(1))
	assert.Expect(second.closeCount).Should(gomega.Equal(1))
	assert.Expect(third.closeCount).Should(gomega.Equal(1))
	chain.Close(context.Background(), nil)
	assert.Expect(first.closeCount).Should(gomega.Equal(2))
	assert.Expect(second.closeCount).Should(gomega.Equal(2))
	assert.Expect(third.closeCount).Should(gomega.Equal(1))
}

type testChainDataplaneServer struct {
	requestCount, closeCount, monitorCount int
}

func (c *testChainDataplaneServer) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	c.requestCount++
	return state.NextDataplaneRequest(ctx, crossConnect)
}

func (c *testChainDataplaneServer) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	c.closeCount++
	return state.NextDataplaneClose(ctx, crossConnect)
}

func (c *testChainDataplaneServer) MonitorMechanisms(empty *empty.Empty, monitorServer dataplane.Dataplane_MonitorMechanismsServer) error {
	c.monitorCount++
	return nil
}

type branchChainDataplaneRequst struct {
	*testChainDataplaneServer
}

func (c *branchChainDataplaneRequst) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	c.requestCount++
	if c.requestCount > 1 {
		return &crossconnect.CrossConnect{}, nil
	}
	return state.NextDataplaneRequest(ctx, crossConnect)
}

func (c *branchChainDataplaneRequst) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	c.closeCount++
	if c.closeCount > 1 {
		return new(empty.Empty), nil
	}
	return state.NextDataplaneClose(ctx, crossConnect)
}
