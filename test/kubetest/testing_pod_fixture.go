package kubetest

import (
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

// TestingPodFixture - Tool for help testing pods
type TestingPodFixture interface {
	DeployNsc(*K8s, *v1.Node, string, time.Duration) *v1.Pod
	DeployNse(*K8s, *v1.Node, string, time.Duration) *v1.Pod
	CheckNsc(*K8s, *v1.Pod) *NSCCheckInfo
}

// VppAgentTestingPodFixture - Creates vpp agent specific testing tool
func VppAgentTestingPodFixture() TestingPodFixture {
	return NewCustomTestingPodFixture(DeployVppAgentNSC, DeployVppAgentICMP, CheckVppAgentNSC)
}

// DefaultTestingPodFixture - Creates default testing tool
func DefaultTestingPodFixture() TestingPodFixture {
	return NewCustomTestingPodFixture(DeployNSC, DeployICMP, CheckNSC)
}

// HealTestingPodFixture - Creates a testing tool specific for healing
func HealTestingPodFixture() TestingPodFixture {
	return NewCustomTestingPodFixture(DeployNSC, DeployICMP, HealNscChecker)
}

// NewCustomTestingPodFixture - Creates a custom testing tool
func NewCustomTestingPodFixture(deployNscFunc, deployNseFunc PodSupplier, checkNscFunc NscChecker) TestingPodFixture {
	gomega.Expect(deployNscFunc).ShouldNot(gomega.BeNil())
	gomega.Expect(deployNseFunc).ShouldNot(gomega.BeNil())
	gomega.Expect(checkNscFunc).ShouldNot(gomega.BeNil())
	return &testingPodFixtureImpl{
		deployNscFunc: deployNscFunc,
		deployNseFunc: deployNseFunc,
		checkNscFunc:  checkNscFunc,
	}
}

type testingPodFixtureImpl struct {
	deployNscFunc, deployNseFunc PodSupplier
	checkNscFunc                 NscChecker
}

//DeployNsc - Deploys network service mesh client with a specific name and node
func (f *testingPodFixtureImpl) DeployNsc(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return f.deployNscFunc(k8s, node, name, timeout)
}

//DeployNse - Deploys network service mesh endpoint with a specific name and node
func (f *testingPodFixtureImpl) DeployNse(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return f.deployNseFunc(k8s, node, name, timeout)
}

//CheckNsc - Perform default check for the client to NSE operations
func (f *testingPodFixtureImpl) CheckNsc(k8s *K8s, nsc *v1.Pod) *NSCCheckInfo {
	return f.checkNscFunc(k8s, nsc)
}
