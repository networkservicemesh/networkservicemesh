package tests

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/runners"
)

func TestOnFailShellTestRunner(t *testing.T) {
	assert := gomega.NewWithT(t)
	runner := runners.NewShellTestRunner("0", &model.TestEntry{
		RunScript:       "12345",
		OnFailScript:    "echo fail",
		ExecutionConfig: &config.ExecutionConfig{},
	}, nil)
	buff := bytes.Buffer{}
	writer := bufio.NewWriter(&buff)
	err := runner.Run(context.TODO(), []string{}, writer)
	assert.Expect(err).ShouldNot(gomega.BeNil())
	assert.Expect(strings.Contains(err.Error(), "12345")).Should(gomega.BeTrue())
	s := buff.String()
	assert.Expect(strings.Contains(s, "Running: echo fail")).Should(gomega.BeTrue())

}

func TestOnFailHasWrongScriptShellTestRunner(t *testing.T) {
	assert := gomega.NewWithT(t)
	runner := runners.NewShellTestRunner("0", &model.TestEntry{
		RunScript:       "12345",
		OnFailScript:    "54321",
		ExecutionConfig: &config.ExecutionConfig{},
	}, nil)
	buff := bytes.Buffer{}
	writer := bufio.NewWriter(&buff)
	err := runner.Run(context.TODO(), []string{}, writer)
	assert.Expect(err).ShouldNot(gomega.BeNil())
	msg := err.Error()
	assert.Expect(strings.Contains(msg, "12345")).Should(gomega.BeTrue())
	assert.Expect(strings.Contains(msg, "54321")).Should(gomega.BeTrue())
	s := buff.String()
	assert.Expect(strings.Contains(s, "Running: 12345")).Should(gomega.BeTrue())
	assert.Expect(strings.Contains(s, "Running: 54321")).Should(gomega.BeTrue())
}

func TestOnFailNotCalledIfRunScriptSucceeded(t *testing.T) {
	assert := gomega.NewWithT(t)
	runner := runners.NewShellTestRunner("0", &model.TestEntry{
		RunScript:       "echo pass",
		OnFailScript:    "echo fail",
		ExecutionConfig: &config.ExecutionConfig{},
	}, nil)
	buff := bytes.Buffer{}
	writer := bufio.NewWriter(&buff)
	err := runner.Run(context.TODO(), []string{}, writer)
	assert.Expect(err).Should(gomega.BeNil())
	s := buff.String()
	assert.Expect(strings.Contains(s, "pass")).Should(gomega.BeTrue())
	assert.Expect(strings.Contains(s, "fail")).Should(gomega.BeFalse())
}
