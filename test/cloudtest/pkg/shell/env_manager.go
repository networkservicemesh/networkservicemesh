package shell

import (
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"math/rand"
	"os"
	"time"
)

// EnvironmentManager - manages enviornment variables.
type EnvironmentManager interface {
	// ProcessEnvironment - process substitute of environment variables with arguments.
	ProcessEnvironment(clusterId, providerName, tempDir string, env []string, extraArgs map[string]string) error
	// GetProcessedEnv - return substituted environment variables
	GetProcessedEnv() []string
	// AddExtraArgs - add argument to map of substitute arguments $(arg) = value
	AddExtraArgs(key, value string)
}

type environmentManager struct {
	processedEnv   []string
	configLocation string
	finalArgs      map[string]string
}

// NewManager - creates a new shell manager
func NewEnvironmentManager() EnvironmentManager {
	return &environmentManager{
	}
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

func NewRandomStr(size int) string {
	var value []byte = make([]byte, size/2)
	_, err := rand.Read(value)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(value)
}

func (em *environmentManager) ProcessEnvironment(clusterId, providerName, tempDir string, env []string, extraArgs map[string]string) error {
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
		randValue := fmt.Sprintf("%v", rand.Intn(1000000))

		randValue30 := NewRandomStr(30)
		randValue10 := NewRandomStr(10)

		args := map[string]string{
			"cluster-name":  clusterId,
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
		"cluster-name":  clusterId,
		"provider-name": providerName,
		"tempdir":       tempDir,
	}

	for k, v := range extraArgs {
		em.finalArgs[k] = v
	}

	return nil
}
