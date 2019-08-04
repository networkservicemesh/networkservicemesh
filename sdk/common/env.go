package common

import (
	"fmt"
	"os"
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

//GetOrDefaultStringListValue returns list of string separated by space or if env variable have not a value returns default value
func (v EnvVar) GetOrDefaultStringListValue(defaultValue []string) []string {
	r := v.StringValue()
	if r == "" {
		return defaultValue
	}
	return strings.Split(r, " ")
}

//GetOrDefaultStringValue returns env value as string or if env variable have not a value returns default value
func (v EnvVar) GetOrDefaultStringValue(defaultValue string) string {
	r := v.StringValue()
	if r == "" {
		return defaultValue
	}
	return r
}

//GetOrDefaultDurationValue returns env value as duration or if env variable have not a value returns default value
func (v EnvVar) GetOrDefaultDurationValue(defaultValue time.Duration) time.Duration {
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
