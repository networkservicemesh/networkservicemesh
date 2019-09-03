package runners

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/shell"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

type goTestRunner struct {
	test    *model.TestEntry
	cmdLine string
	envMgr  shell.EnvironmentManager
}

func (runner *goTestRunner) Run(timeoutCtx context.Context, env []string, writer *bufio.Writer) error {
	logger := func(s string) {}
	cmdEnv := append(runner.envMgr.GetProcessedEnv(), env...)
	_, err := utils.RunCommand(timeoutCtx, runner.cmdLine, logger, writer, cmdEnv, map[string]string{}, false)

	// If go test finished with error we have to clean up created namespaces manually
	if err != nil {
		cleanupCreatedNamespaces(cmdEnv, writer)
	}
	return err
}

func (runner *goTestRunner) GetCmdLine() string {
	return runner.cmdLine
}

// NewGoTestRunner - creates go test runner
func NewGoTestRunner(ids string, test *model.TestEntry, timeout int64) TestRunner {
	cmdLine := fmt.Sprintf("go test %s -test.timeout %ds -count 1 --run \"^(%s)$\\\\z\" --tags \"%s\" --test.v",
		test.ExecutionConfig.PackageRoot, timeout, test.Name, test.Tags)

	envMgr := shell.NewEnvironmentManager()
	_ = envMgr.ProcessEnvironment(ids, "gotest", os.TempDir(), test.ExecutionConfig.Env, map[string]string{})

	return &goTestRunner{
		test:    test,
		cmdLine: cmdLine,
		envMgr:  envMgr,
	}
}

func cleanupCreatedNamespaces(env []string, writer *bufio.Writer) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger := func(s string) {}

	curNamespace := "default"
	for _, k := range env {
		key, value, err := utils.ParseVariable(k)
		if err == nil {
			if key == "NSM_NAMESPACE" {
				curNamespace = value
			}
		}
	}

	cCmd := "kubectl get ns -o custom-columns=NAME:.metadata.name"
	namespaces, err := utils.RunCommand(timeoutCtx, cCmd, logger, writer, env, map[string]string{}, true)
	nss := strings.Split(namespaces, "\n")

	if err == nil {
		for _, ns := range nss {
			if strings.Contains(ns, curNamespace) {
				cCmd = fmt.Sprintf("kubectl delete ns %s", ns)
				_, err = utils.RunCommand(timeoutCtx, cCmd, logger, writer, env, map[string]string{}, true)
				if err != nil {
					logrus.Warnf("Namespace %s was not cleared", ns)
				}
			}
		}
	}
}
