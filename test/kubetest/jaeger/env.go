package jaeger

import (
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/utils"
)

const (
	jaegerPrefix = "JAEGER"
	//StoreJaegerTraces env variable represents boolean, means store jaeger traces as files
	StoreJaegerTraces utils.EnvVar = "STORE_JAEGER_TRACES"
	//JaegerRestAPIPort means port of ingester api server
	JaegerRestAPIPort utils.EnvVar = "JAEGER_REST_API_PORT"
)

var jaegerEnv []v1.EnvVar

func init() {
	jaegerEnv = processUserEnv()
}

//ShouldStoreJaegerTraces means store jaeger traces as files
func ShouldStoreJaegerTraces() bool {
	return StoreJaegerTraces.GetBooleanOrDefault(false)
}

//GetJaegerRestAPIPort returns jaeger API port
func GetJaegerRestAPIPort() int {
	return JaegerRestAPIPort.GetIntOrDefault(16686)
}

//DefaultEnvValues returns default jaeger env values
func DefaultEnvValues() map[string]string {
	return map[string]string{
		"JAEGER_AGENT_HOST": "jaeger",
		"JAEGER_AGENT_PORT": "6831",
		"JAEGER_API_PORT":   "16686",
	}
}

//Env converts user's jaeger env to []v1.EnvVar
func Env() []v1.EnvVar {
	return jaegerEnv
}

func processUserEnv() []v1.EnvVar {
	envs := os.Environ()
	envMap := map[string]string{}
	defaultEnvs := DefaultEnvValues()
	result := []v1.EnvVar{}
	for _, env := range envs {
		if strings.HasPrefix(env, jaegerPrefix) {
			envParts := strings.Split(env, "=")
			if len(envParts) < 2 {
				continue
			}
			envMap[envParts[0]] = envParts[1]
		}
	}
	for k, v := range defaultEnvs {
		if envMap[k] == "" {
			envMap[k] = v
		}
	}
	for k, v := range envMap {
		result = append(result, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return result
}
