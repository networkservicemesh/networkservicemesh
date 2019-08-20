package fanout

import "time"

const (
	maxIPCount          = 15
	maxTimeout          = 2 * time.Second
	defaultTimeout      = 30 * time.Second
	defaultExpire       = 10 * time.Second
	minDialTimeout      = 1 * time.Second
	maxDialTimeout      = 30 * time.Second
	readTimeout         = 2 * time.Second
	tcptlc              = "tcp-tls"
	cumulativeAvgWeight = 4
)
