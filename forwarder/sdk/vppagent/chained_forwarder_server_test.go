package vppagent

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
)

type testChainForwarderServer struct {
	requestCount, closeCount int
}

func (c *testChainForwarderServer) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	c.requestCount++
	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	return next.Request(ctx, crossConnect)
}

func (c *testChainForwarderServer) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	c.closeCount++
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConnect)
}

type branchChainForwarderRequst struct {
	requestCount, closeCount, monitorCount int
}

func (c *branchChainForwarderRequst) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	c.requestCount++
	if c.requestCount > 1 {
		return &crossconnect.CrossConnect{}, nil
	}
	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	return next.Request(ctx, crossConnect)
}

func (c *branchChainForwarderRequst) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	c.closeCount++
	if c.closeCount > 1 {
		return new(empty.Empty), nil
	}
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConnect)
}

func TestBasicForwarderChain(t *testing.T) {
	gomega.RegisterTestingT(t)
	assert := gomega.NewWithT(t)
	first := &testChainForwarderServer{}
	second := &testChainForwarderServer{}

	chain := ChainOf(first, second)
	_, _ = chain.Request(context.Background(), nil)
	assert.Expect(first.requestCount).Should(gomega.Equal(1))
	assert.Expect(second.requestCount).Should(gomega.Equal(1))
	assert.Expect(first.closeCount).Should(gomega.Equal(0))
	assert.Expect(second.closeCount).Should(gomega.Equal(0))
	_, _ = chain.Close(context.Background(), nil)
	assert.Expect(first.requestCount).Should(gomega.Equal(1))
	assert.Expect(second.requestCount).Should(gomega.Equal(1))
	assert.Expect(first.closeCount).Should(gomega.Equal(1))
	assert.Expect(second.closeCount).Should(gomega.Equal(1))
}

func TestBranchForwarderChain(t *testing.T) {
	gomega.RegisterTestingT(t)
	assert := gomega.NewWithT(t)
	first := &testChainForwarderServer{}
	second := &branchChainForwarderRequst{}
	third := &testChainForwarderServer{}
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
	_, _ = chain.Close(context.Background(), nil)
	assert.Expect(first.closeCount).Should(gomega.Equal(1))
	assert.Expect(second.closeCount).Should(gomega.Equal(1))
	assert.Expect(third.closeCount).Should(gomega.Equal(1))
	_, _ = chain.Close(context.Background(), nil)
	assert.Expect(first.closeCount).Should(gomega.Equal(2))
	assert.Expect(second.closeCount).Should(gomega.Equal(2))
	assert.Expect(third.closeCount).Should(gomega.Equal(1))
}
