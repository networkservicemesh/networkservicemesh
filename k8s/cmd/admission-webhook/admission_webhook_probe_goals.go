package main

import "fmt"

const (
	keyPairLoaded = 1 << iota
	serverStarted
	done = keyPairLoaded | serverStarted
)

type admissionWebhookGoals struct {
	state int8
}

func (g *admissionWebhookGoals) SetKeyPairLoaded() {
	g.state |= keyPairLoaded
}

func (g *admissionWebhookGoals) SetServerStarted() {
	g.state |= serverStarted
}

func (g *admissionWebhookGoals) IsComplete() bool {
	return g.state == done
}

func (g *admissionWebhookGoals) TODO() string {
	return fmt.Sprintf("Key pair loaded: %v, Server started: %v,",
		g.state&keyPairLoaded == 1,
		g.state&serverStarted == 1)
}