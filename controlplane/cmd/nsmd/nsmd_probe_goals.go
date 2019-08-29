package main

import (
	"fmt"
)

const (
	nsmServerReady = 1 << iota
	dataplaneServerReady
	publicListenerReady
	serverAPIReady
	done = nsmServerReady | dataplaneServerReady | publicListenerReady | serverAPIReady
)

type nsmdProbeGoals struct {
	state int8
}

func (g *nsmdProbeGoals) Status() string {
	return fmt.Sprintf("NSM Server is ready: %v, Dataplane server is ready: %v, Public listener is ready: %v, Server API is ready: %v",
		g.state&nsmServerReady > 0,
		g.state&dataplaneServerReady > 0,
		g.state&publicListenerReady > 0,
		g.state&serverAPIReady > 0,
	)
}

func (g *nsmdProbeGoals) SetNsmServerReady() {
	g.state |= nsmServerReady
}
func (g *nsmdProbeGoals) SetDataplaneServerReady() {
	g.state |= dataplaneServerReady
}
func (g *nsmdProbeGoals) SetPublicListenerReady() {
	g.state |= publicListenerReady
}
func (g *nsmdProbeGoals) SetServerAPIReady() {
	g.state |= serverAPIReady
}
func (g *nsmdProbeGoals) IsComplete() bool {
	return g.state == done
}
