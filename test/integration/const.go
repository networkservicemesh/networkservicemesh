package nsmd_integration_tests

import "time"

const (
	ciDelayCoefficient = 1
	defaultTimeout     = 2 * time.Minute * ciDelayCoefficient
	fastTimeout        = defaultTimeout / 5
)
