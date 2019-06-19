package shell

import (
	"bufio"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"strings"
	"time"
)

// Manager - allow to perform shell command executions with variable and parameter substitutions.
type Manager interface {
	// GetConfigLocation - detect if KUBECONFIG variable is passed and return its value.
	GetConfigLocation() string
	// RunCmd - execute a command, operation with extra env
	RunCmd(context context.Context, operation string, script [] string, env [] string) error
	// ProcessEnvironment - process substitute of environment variables with arguments.
	ProcessEnvironment(extraArgs map[string]string) error
	// PrintEnv - print environment variables into string
	PrintEnv(processedEnv []string) string
	// PrintArgs - print arguments to string
	PrintArgs() string
	// GetProcessedEnv - return substituted environment variables
	GetProcessedEnv() []string
	// AddExtraArgs - add argument to map of substitute arguments $(arg) = value
	AddExtraArgs(key, value string)
}

type shellInterface struct {
	root           string
	id             string
	config         *config.ClusterProviderConfig
	processedEnv   []string
	manager        execmanager.ExecutionManager
	params         providers.InstanceOptions
	configLocation string
	finalArgs      map[string]string
}

func (si *shellInterface) AddExtraArgs(key, value string) {
	si.finalArgs[key] = value
}

func (si *shellInterface) GetProcessedEnv() []string {
	return si.processedEnv
}

// NewManager - creates a new shell manager
func NewManager(manager execmanager.ExecutionManager, id, root string, config *config.ClusterProviderConfig,
	params providers.InstanceOptions) Manager {
	return &shellInterface{
		manager: manager,
		root:    root,
		id:      id,
		config:  config,
		params:  params,
	}
}

func (si *shellInterface) GetConfigLocation() string {
	return si.configLocation
}

// Run command in context and add appropriate execution output file.
func (si *shellInterface) RunCmd(context context.Context, operation string, script, env []string) error {
	_, fileRef, err := si.manager.OpenFile(si.id, operation)
	if err != nil {
		logrus.Errorf("failed to %s system for testing of cluster %s %v", operation, si.config.Name, err)
		return err
	}

	defer func() { _ = fileRef.Close() }()

	writer := bufio.NewWriter(fileRef)

	for _, cmd := range script {
		if strings.TrimSpace(cmd) == "" {
			continue
		}

		cmdEnv := append(si.processedEnv, env...)
		printableEnv := si.PrintEnv(env)

		_, _ = writer.WriteString(fmt.Sprintf("%s: %v\nENV={\n%v\n}\n", operation, cmd, printableEnv))
		_ = writer.Flush()

		logrus.Infof("%s: %s => %s", operation, si.id, cmd)

		if err := utils.RunCommand(context, si.id, cmd, operation, writer, cmdEnv, si.finalArgs); err != nil {
			_, _ = writer.WriteString(fmt.Sprintf("error running command: %v\n", err))
			_ = writer.Flush()
			return err
		}
	}
	return nil
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

func (si *shellInterface) ProcessEnvironment(extraArgs map[string]string) error {
	environment := map[string]string{}

	for _, k := range os.Environ() {
		key, value, err := utils.ParseVariable(k)
		if err != nil {
			return err
		}
		environment[key] = value
	}

	today := time.Now()

	todayYear := fmt.Sprintf("%d", today.Year())
	todayMonth := fmt.Sprintf("%d", today.Month())
	todayDay := fmt.Sprintf("%d", today.Day())
	for _, rawVarName := range si.config.Env {
		varName, varValue, err := utils.ParseVariable(rawVarName)
		if err != nil {
			return err
		}
		randValue := fmt.Sprintf("%v", rand.Intn(1000000))
		uuidValue := uuid.New().String()
		uuidValue30 := uuid.New().String()[:30]
		uuidValue10 := uuid.New().String()[:10]

		args := map[string]string{
			"cluster-name":  si.id,
			"provider-name": si.config.Name,
			"random":        randValue,
			"uuid":          uuidValue,
			"uuid30":        uuidValue30,
			"uuid10":        uuidValue10,
			"tempdir":       si.root,
			"year":          todayYear,
			"month":         todayMonth,
			"date":          fmt.Sprintf("%s-%s-%s", todayYear, todayMonth, todayDay),
			"day":           todayDay,
		}

		for k, v := range extraArgs {
			args[k] = v
		}

		varValue, err = utils.SubstituteVariable(varValue, environment, args)
		if err != nil {
			return err
		}

		// Now we need to parse  line and replace all ${VAR_NAME} with real and processed environment variables.

		if varName == "KUBECONFIG" {
			si.configLocation = varValue
		}

		environment[varName] = varValue
		si.processedEnv = append(si.processedEnv, fmt.Sprintf("%s=%s", varName, varValue))
	}

	si.finalArgs = map[string]string{
		"cluster-name":  si.id,
		"provider-name": si.config.Name,
		"tempdir":       si.root,
	}

	for k, v := range extraArgs {
		si.finalArgs[k] = v
	}

	return nil
}
