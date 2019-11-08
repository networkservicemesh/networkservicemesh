package shell

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

// Manager - allow to perform shell command executions with variable and parameter substitutions.
type Manager interface {
	// GetConfigLocation - detect if KUBECONFIG variable is passed and return its value.
	GetConfigLocation() string
	// RunCmd - execute a command, operation with extra env
	RunCmd(context context.Context, operation string, script []string, env []string) (string, error)
	// RunRead - execute a command, operation with extra env and read response into variable
	RunRead(context context.Context, operation string, script []string, env []string) (string, error)
	// PrintEnv - print environment variables into string
	PrintEnv(processedEnv []string) string
	// PrintArgs - print arguments to string
	PrintArgs() string
	EnvironmentManager
}

type shellInterface struct {
	id      string
	config  *config.ClusterProviderConfig
	manager execmanager.ExecutionManager
	params  providers.InstanceOptions
	environmentManager
}

// NewManager - creates a new shell manager
func NewManager(manager execmanager.ExecutionManager, id string, config *config.ClusterProviderConfig,
	params providers.InstanceOptions) Manager {
	return &shellInterface{
		manager: manager,
		id:      id,
		config:  config,
		params:  params,
	}
}

// RunCmd -  command in context and add appropriate execution output file.
func (si *shellInterface) RunCmd(context context.Context, operation string, script, env []string) (string, error) {
	fileName, _, err := si.runCmd(context, operation, script, env, false)
	return fileName, err
}

// Run command in context and add appropriate execution output file.
func (si *shellInterface) RunRead(context context.Context, operation string, script, env []string) (string, error) {
	_, response, err := si.runCmd(context, operation, script, env, true)
	return response, err
}
func (si *shellInterface) runCmd(context context.Context, operation string, script, env []string, returnResult bool) (string, string, error) {
	fileName, fileRef, err := si.manager.OpenFile(si.id, operation)
	if err != nil {
		logrus.Errorf("failed to %s system for testing of cluster %s %v", operation, si.config.Name, err)
		return fileName, "", err
	}

	defer func() { _ = fileRef.Close() }()

	writer := bufio.NewWriter(fileRef)

	finalOut := ""
	for _, cmd := range script {
		if strings.TrimSpace(cmd) == "" {
			continue
		}

		cmdEnv := append(si.processedEnv, env...)
		printableEnv := si.PrintEnv(env)

		_, _ = writer.WriteString(fmt.Sprintf("%s: %v\nENV={\n%v\n}\n", operation, cmd, printableEnv))
		_ = writer.Flush()

		logrus.Infof("%s: %s => %s", operation, si.id, cmd)

		logger := func(s string) {
			// logrus.Infof("%s: %s -> %v", si.id, operation, s)
		}
		stdOut, err := utils.RunCommand(context, cmd, "", logger, writer, cmdEnv, si.finalArgs, returnResult)
		if err != nil {
			_, _ = writer.WriteString(fmt.Sprintf("error running command: %v\n", err))
			_ = writer.Flush()
			return fileName, "", err
		}
		if returnResult {
			finalOut += stdOut
		}
	}
	return fileName, finalOut, nil
}
func (si *shellInterface) PrintEnv(processedEnv []string) string {
	printableEnv := strings.Builder{}
	for _, cmdEnvValue := range processedEnv {
		varName, varValue, _ := utils.ParseVariable(cmdEnvValue)

		if !si.params.NoMaskParameters {
			// We need to check if value contains or not some of check env variables and replace their values for safity
			for _, ce := range si.config.EnvCheck {
				envValue := os.Getenv(ce)
				varValue = strings.Replace(varValue, envValue, "****", -1)
			}
		}
		_, _ = printableEnv.WriteString(fmt.Sprintf("%s=%s\n", varName, varValue))
	}
	return printableEnv.String()
}
func (si *shellInterface) PrintArgs() string {
	printableEnv := strings.Builder{}

	_, _ = printableEnv.WriteString("Arguments:\n")

	for varName, varValue := range si.finalArgs {
		if !si.params.NoMaskParameters {
			// We need to check if value contains or not some of check env variables and replace their values for safity
			for _, ce := range si.config.EnvCheck {
				envValue := os.Getenv(ce)
				varValue = strings.Replace(varValue, envValue, "****", -1)
			}
		}
		_, _ = printableEnv.WriteString(fmt.Sprintf("%s=%s\n", varName, varValue))
	}
	return printableEnv.String()
}
