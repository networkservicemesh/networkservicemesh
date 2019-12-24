package nsmd_integration_tests

import "time"

const (
	defaultTimeout  = 2 * time.Minute
	fastTimeout     = defaultTimeout / 5
	nscDefaultName  = "nsc"
	icmpDefaultName = "icmp-responder"
	nscCount        = 10
	nscMaxCount     = 20
)
