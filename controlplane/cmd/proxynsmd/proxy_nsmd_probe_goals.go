package main

import "fmt"

const (
	publicListenerReady = 1 << iota
	serverAPIReady
	done = publicListenerReady | serverAPIReady
)

type proxyNsmdProbeGoals struct {
	state int8
}

func (g *proxyNsmdProbeGoals) TODO() string {
	return fmt.Sprintf("Public listener is ready: %v, Server api is ready: %v",
		g.state&publicListenerReady == 1,
		g.state&serverAPIReady == 1,
	)
}
func (g *proxyNsmdProbeGoals) SetPublicListenerReady() {
	g.state |= publicListenerReady
}
func (g *proxyNsmdProbeGoals) SetServerAPIReady() {
	g.state |= serverAPIReady
}
func (g *proxyNsmdProbeGoals) IsComplete() bool {
	return g.state == done
}
