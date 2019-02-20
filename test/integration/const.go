package nsmd_integration_tests

import "time"

const (
	ciDelayCoefficient = 1
	defaultTimeout     = 60 * time.Second * ciDelayCoefficient
	fastTimeout        = defaultTimeout / 5
)
