// Package integration contains kubernetes specific NSM tests
package integration

import "time"

const (
	defaultTimeout  = 5 * time.Minute
	fastTimeout     = defaultTimeout / 10
	nscDefaultName  = "nsc"
	icmpDefaultName = "icmp-responder"
	nscCount        = 10
	nscMaxCount     = 20
)
