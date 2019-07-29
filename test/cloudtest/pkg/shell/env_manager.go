package shell

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

// EnvironmentManager - manages environment variables.
type EnvironmentManager interface {
	// ProcessEnvironment - process substitute of environment variables with arguments.
	ProcessEnvironment(clusterID, providerName, tempDir string, env []string, extraArgs map[string]string) error
	// GetProcessedEnv - return substituted environment variables
	GetProcessedEnv() []string
	// AddExtraArgs - add argument to map of substitute arguments $(arg) = value
	AddExtraArgs(key, value string)
	//
	GetArguments() map[string]string
}

type environmentManager struct {
	processedEnv   []string
	configLocation string
	finalArgs      map[string]string
}

func (em *environmentManager) GetArguments() map[string]string {
	return em.finalArgs
}

// NewEnvironmentManager - creates a new environment variable manager
func NewEnvironmentManager() EnvironmentManager {
	return &environmentManager{}
}

func (em *environmentManager) AddExtraArgs(key, value string) {
	em.finalArgs[key] = value
}

func (em *environmentManager) GetProcessedEnv() []string {
	return em.processedEnv
}

func (em *environmentManager) GetConfigLocation() string {
	return em.configLocation
}

func (em *environmentManager) ProcessEnvironment(clusterID, providerName, tempDir string, env []string, extraArgs map[string]string) error {
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
	for _, rawVarName := range env {
		varName, varValue, err := utils.ParseVariable(rawVarName)
		if err != nil {
			return err
		}
		randNum, err := rand.Int(rand.Reader, big.NewInt(1000000))
		randValue := ""
		if err != nil {
			logrus.Errorf("Error during random number generation %v", err)
		} else {
			randValue = fmt.Sprintf("%v", randNum)
		}

		randValue30 := utils.NewRandomStr(30)
		randValue10 := utils.NewRandomStr(10)

		args := map[string]string{
			"cluster-name":  clusterID,
			"provider-name": providerName,
			"random":        randValue,
			"uuid":          uuid.New().String(),
			"rands30":       randValue30,
			"rands10":       randValue10,
			"tempdir":       tempDir,
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
			em.configLocation = varValue
		}

		environment[varName] = varValue
		em.processedEnv = append(em.processedEnv, fmt.Sprintf("%s=%s", varName, varValue))
	}

	em.finalArgs = map[string]string{
		"cluster-name":  clusterID,
		"provider-name": providerName,
		"tempdir":       tempDir,
	}

	for k, v := range extraArgs {
		em.finalArgs[k] = v
	}
	return nil
}
