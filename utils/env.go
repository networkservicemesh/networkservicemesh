package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

//EnvVar provides API for access to env variable
type EnvVar string

func (v EnvVar) String() string {
	return fmt.Sprintf("%v:%v", v.Name(), v.StringValue())
}

//StringValue returns value of env variable as string
func (v EnvVar) StringValue() string {
	return os.Getenv(v.Name())
}

//GetStringListValueOrDefault returns list of string separated by space or if env variable have not a value returns default value
func (v EnvVar) GetStringListValueOrDefault(defaultValues ...string) []string {
	r := v.StringValue()
	if r == "" {
		return defaultValues
	}
	return strings.Split(r, " ")
}

//GetStringOrDefault returns env value as string or if env variable have not a value returns default value
func (v EnvVar) GetStringOrDefault(defaultValue string) string {
	r := v.StringValue()
	if r == "" {
		return defaultValue
	}
	return r
}

//GetBooleanOrDefault returns env value as string or if env variable have not a value returns default value
func (v EnvVar) GetBooleanOrDefault(defaultValue bool) bool {
	str := v.StringValue()
	if v, err := strconv.ParseBool(str); err == nil {
		return v
	}
	return defaultValue
}

//GetOrDefaultDuration returns env value as duration or if env variable have not a value returns default value
func (v EnvVar) GetOrDefaultDuration(defaultValue time.Duration) time.Duration {
	val := v.StringValue()
	if val == "" {
		return defaultValue
	}
	r, err := time.ParseDuration(val)
	if err != nil {
		logrus.Errorf("Can't parse %v as Duration, reason: %v", v.String(), err)
		return defaultValue
	}
	return r
}

//Name returns emv variable name
func (v EnvVar) Name() string {
	return string(v)
}
